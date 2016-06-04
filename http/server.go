package httpsvc

func Serve(t Transport) {
	// @TODO: Actually implement proper termination
	<-make(chan struct{}, 0)
}
