proto-generate:
	protoc -I ./api/proto \
		--go_out=./api/gen/$(SVC) --go_opt=paths=source_relative \
		--go-grpc_out=./api/gen/$(SVC) --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=./api/gen/$(SVC) --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=./api/swagger ./api/proto/$(SVC).proto

# create-table:
# 	migrate create -ext sql -dir internal/$(SVC)/migrations/ -seq $(SEQ)

create-table:
	migrate create -ext sql -dir migrations/$(SVC) -seq $(SEQ)

ifndef SVC
	$(error add service name using SVC=...)
endif
ifndef SEQ
	$(error add migration name using SEQ=...)
endif

