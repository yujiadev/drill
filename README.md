myapp/
├── cmd/
│   └── myapp/
│       └── main.go       # Main application entry point
├── internal/
│   ├── app/              # Application-specific code
│   │   ├── config.go     # Configuration handling
│   │   └── runner.go    # Main application logic
│   └── pkg/              # Internal packages
│       └── utils/        # Utility functions
│           └── helpers.go
├── pkg/
│   └── publicpkg/        # Publicly importable packages
│       └── lib.go
├── go.mod                # Go module definition
├── go.sum                # Dependency checksums
├── Makefile              # Common build tasks
├── README.md             # Project documentation
├── .gitignore            # Git ignore rules
├── configs/              # Configuration templates/defaults
│   └── config.yaml
├── scripts/              # Scripts for build/install/etc.
│   └── install.sh
└── test/                 # Additional test files
    └── integration/
        └── integration_test.go