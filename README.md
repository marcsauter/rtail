
	protoc -I pkg/pb/ pkg/pb/rtail.proto --go_out=plugins=grpc:pkg/pb
