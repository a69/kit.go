package proto

//go:generate protoc proto_test.proto --go_out=. --go_opt=Mproto_test.proto=github.com/a69/kit.go/transport/http/proto --go_opt=paths=source_relative
//go:generate mv proto_test.pb.go proto_pb_test.go
