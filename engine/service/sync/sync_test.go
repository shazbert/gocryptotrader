package sync

import (
	"fmt"
	"log"
	"runtime"

	uuid "github.com/satori/go.uuid"

	"github.com/thrasher-/gocryptotrader/config"

	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/exchanges/asset"
	"github.com/thrasher-/gocryptotrader/exchanges/binance"
)

func gen(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		for _, n := range nums {
			out <- n
		}
		close(out)
	}()
	return out
}

func sq(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		for n := range in {
			out <- n * n
		}
		close(out)
	}()
	return out
}

func TestFunctions(t *testing.T) {
	//MEOW
	for n := range sq(gen(2, 3)) {
		fmt.Println(n)
	}
}

func TestGetManager(t *testing.T) {
	m := GetManager()

	b := binance.Binance{}
	b.SetDefaults()
	b.Verbose = false
	c := config.GetConfig()
	err := c.LoadConfig("../../" + config.ConfigTestFile)
	if err != nil {
		t.Fatal("COWS", err)
	}

	conf, err := c.GetExchangeConfig("Binance")
	if err != nil {
		t.Fatal("COWS", err)
	}

	err = b.Setup(conf)
	if err != nil {
		t.Fatal("COWS", err)
	}

	// fmt.Println(len(b.GetAvailablePairs(asset.Spot)))
	// os.Exit(1)

	var cats []func() error
	for _, meow := range b.GetAvailablePairs(asset.Spot) {
		meow := meow
		fn := func() error {
			// log.Debugln(log.ExchangeSys, meow.String())
			// data, err := b.UpdateTicker(meow, asset.Spot)
			// if err != nil {
			// 	return err
			// }

			// log.Debugln(log.ExchangeSys, data.Last)
			// return nil
			log.Printf("Fetching data for %s\n", meow)
			return nil
		}

		cats = append(cats, fn)
	}

	serviceID, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}
	err = m.NewSyncGroup(serviceID, cats)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(20 * time.Second)

	PrintMemUsage()
	fmt.Println("trying to shutdowns")
	err = m.Shutdown()
	if err != nil {
		t.Error(err)
	}

	fmt.Println(m)
}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func TestThings(t *testing.T) {
	var b interface{}
	b = true
	if b.(bool) {
		fmt.Println("YAY")
	}
}

func BenchmarkBenchy(b *testing.B) {
	for n := 0; n < b.N; n++ {
		fmt.Printf("%s", "Hello,World")
	}
}
