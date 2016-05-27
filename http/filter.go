package httpsvc

type Filter func(Request, Service) Response
