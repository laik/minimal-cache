package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Bowery/prompt"
	"github.com/laik/minimal-cache/api"
	"github.com/laik/minimal-cache/insecure"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/naming"
)

var errWatcherClose = errors.New("watcher has been closed")

const connectTimeout = 200 * time.Millisecond

const prefix = "minimal >"

var prmpt = ""

// NewPseudoResolver creates a new pseudo resolver which returns fixed addrs.
func NewPseudoResolver(addrs []string) naming.Resolver {
	return &pseudoResolver{addrs}
}

type pseudoResolver struct {
	addrs []string
}

func (r *pseudoResolver) Resolve(target string) (naming.Watcher, error) {
	w := &pseudoWatcher{
		updatesChan: make(chan []*naming.Update, 1),
	}
	updates := []*naming.Update{}
	for _, addr := range r.addrs {
		updates = append(updates, &naming.Update{Op: naming.Add, Addr: addr})
	}
	w.updatesChan <- updates
	return w, nil
}

// This watcher is implemented based on ipwatcher below
// https://github.com/grpc/grpc-go/blob/30fb59a4304034ce78ff68e21bd25776b1d79488/naming/dns_resolver.go#L151-L171
type pseudoWatcher struct {
	updatesChan chan []*naming.Update
}

func (w *pseudoWatcher) Next() ([]*naming.Update, error) {
	us, ok := <-w.updatesChan
	if !ok {
		return nil, errWatcherClose
	}
	return us, nil
}

func (w *pseudoWatcher) Close() {
	close(w.updatesChan)
}

//CLI allows users to interact with a server.
type CLI struct {
	printer *printer
	term    *prompt.Terminal
	conn    *grpc.ClientConn
	client  api.MinimalCacheClient
}

//Run runs a new CLI.
func Run(hostPorts string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	hostPortLength := strings.Count(hostPorts, ",")
	var firstAddr string
	var dialAddrs = make([]string, 0, hostPortLength)
	for i, hostPort := range strings.Split(hostPorts, ",") {
		// dialAddrs = append(dialAddrs, fmt.Sprintf("passthrough://%s/%s", hostPort, hostPort))
		dialAddrs = append(dialAddrs, hostPort)
		if i == 0 {
			firstAddr = hostPort
			continue
		}
		dialAddrs = append(dialAddrs, hostPort)
	}

	balanceAddrs := grpc.WithBalancer(grpc.RoundRobin(NewPseudoResolver(dialAddrs)))

	conn, err := grpc.DialContext(
		ctx,
		firstAddr,
		balanceAddrs,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(insecure.CertPool, "")),
	)

	if err != nil {
		return fmt.Errorf("could not dial %s: %v", balanceAddrs, err)
	}

	term, err := prompt.NewTerminal()
	if err != nil {
		return fmt.Errorf("could not create a terminal: %v", err)
	}

	prmpt = fmt.Sprintf("(%s) %s", hostPorts, prefix)

	c := &CLI{
		printer: newPrinter(os.Stdout),
		term:    term,
		client:  api.NewMinimalCacheClient(conn),
		conn:    conn,
	}

	defer func() {
		err = c.Close()
	}()

	c.run()

	return nil
}

//Close closes the CLI.
func (c *CLI) Close() error {
	if err := c.printer.Close(); err != nil {
		return err
	}
	if err := c.conn.Close(); err != nil {
		return err
	}
	return c.term.Close()
}

func (c *CLI) run() {
	c.printer.printLogo()
	for {
		input, err := c.term.GetPrompt(prmpt)
		if err != nil {
			if err == prompt.ErrCTRLC || err == prompt.ErrEOF {
				break
			}
			c.printer.printError(err)
			continue
		}
		if input == "" {
			continue
		}
		req := &api.ExecuteCommandRequest{Command: input}
		if resp, err := c.client.ExecuteCommand(context.Background(), req); err != nil {
			c.printer.printError(err)
		} else {
			c.printer.printResponse(resp)
		}
	}
	c.printer.println("Bye!")
}
