package server

// Serve starts the gromit server
func Serve(ca string, cert string, key string) {
	a := App{}
	a.Init(ca)

	a.Run(":443", cert, key)
}
