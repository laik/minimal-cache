package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Bowery/prompt"
	"github.com/laik/minimal-cache/api"
	"github.com/laik/minimal-cache/insecure"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const connectTimeout = 200 * time.Millisecond

const prefix = "minimal >"

var prmpt = ""

//CLI allows users to interact with a server.
type CLI struct {
	printer *printer
	term    *prompt.Terminal
	conn    *grpc.ClientConn
	client  api.MinimalCacheClient
}

//Run runs a new CLI.
func Run(hostPort string) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	dialAddr := fmt.Sprintf("passthrough://%s/%s", hostPort, hostPort)
	conn, err := grpc.DialContext(
		ctx,
		dialAddr,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(insecure.CertPool, "")),
	)
	if err != nil {
		return fmt.Errorf("could not dial %s: %v", hostPort, err)
	}

	term, err := prompt.NewTerminal()
	if err != nil {
		return fmt.Errorf("could not create a terminal: %v", err)
	}

	prmpt = fmt.Sprintf("(%s) %s", hostPort, prefix)

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
