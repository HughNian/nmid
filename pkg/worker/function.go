package worker

import cli "github.com/HughNian/nmid/pkg/client"

type Job interface {
	GetResponse() *Response
	ParseParams(params []byte)
	GetParams() []byte
	GetParamsMap() map[string]interface{}
	ClientCall(serverAddr, funcName string, params map[string]interface{}, respHandler func(resp *cli.Response))
}

type JobFunc func(Job) ([]byte, error)

type Function struct {
	Func     JobFunc
	FuncName string
}

func NewFunction(jf JobFunc, fname string) *Function {
	return &Function{
		Func:     jf,
		FuncName: fname,
	}
}
