module github.com/luxfi/transfer

go 1.26.1

require (
	github.com/go-chi/chi/v5 v5.2.5
	github.com/go-chi/cors v1.2.2
	github.com/luxfi/crypto v1.17.39
	github.com/luxfi/fhe v0.0.0
	github.com/rs/zerolog v1.34.0
)

require (
	github.com/ALTree/bigfloat v0.2.0 // indirect
	github.com/cloudflare/circl v1.6.3 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/luxfi/lattice/v7 v7.0.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/exp v0.0.0-20260212183809-81e46e3db34a // indirect
	golang.org/x/sys v0.41.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/luxfi/crypto => ../crypto
	github.com/luxfi/fhe => ../fhe
)
