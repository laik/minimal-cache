package command

type Reply interface {
	Val() interface{}
}

type ErrReply struct {
	Message error
}

func (this *ErrReply) Val() interface{} {
	return this.Message
}

type OkReply struct{}

func (this *OkReply) Val() interface{} {
	return nil
}

type StringReply struct {
	Message string
}

func (this *StringReply) Val() interface{} {
	return this.Message
}

type NilReply struct{}

func (this *NilReply) Val() interface{} {
	return nil
}

type SliceReply struct {
	Message []string
}

func (this *SliceReply) Val() interface{} {
	return this.Message
}
