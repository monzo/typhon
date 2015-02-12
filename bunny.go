package bunny

type BunnyServer interface {
	Run()
	RegisterEndpoint()
}
