package worker

type Job interface {
	GetResponse() *Response
	ParseParams(params []byte)
	GetParams() []byte
	GetParamsMap() map[string]interface{}
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
