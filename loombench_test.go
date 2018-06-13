package main

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

// 	"github.com/loomnetwork/go-loom"
// 	"github.com/loomnetwork/go-loom/auth"
// 	"github.com/loomnetwork/go-loom/client"
// )

func init() {
	// cobra.OnInitialize(initConfig)
	flag.StringVar(&config, "config", "", "config file")
	flag.Parse()
	fmt.Println("Config From Init: ", config)
}

func TestMain(m *testing.M) {
	// gin.SetMode(gin.TestMode)
	// flag.Parse()
	fmt.Println("Hello from TestMain.loombench_test.go")
	os.Exit(m.Run())
}

func BenchmarkLoom(b *testing.B) {
	fmt.Printf("Config: '%s'\n", config)
	for n := 0; n < b.N; n++ {
		_ = 1 + 1
	}
}
