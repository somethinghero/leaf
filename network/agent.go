package network

//Agent agent interface
type Agent interface {
	Run()
	OnClose()
}
