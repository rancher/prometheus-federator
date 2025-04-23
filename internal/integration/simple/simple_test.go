package simple_test

import (
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
)

var _ = BeforeSuite(func() {
	fmt.Println("Setup node")
})

var counter = 0
var counterMu sync.Mutex

var _ = Describe("Top level node", func() {

	BeforeEach(OncePerOrdered, func() {
		counterMu.Lock()
		defer counterMu.Unlock()
		time.Sleep(time.Millisecond * 50)
		counter++
		fmt.Println("BeforeEach Top level node", counter)
	})

	var _ = Describe("Ordered Node 1", Ordered, func() {
		When("A", func() {
			It("Test 1", func() {
				time.Sleep(1 * time.Second)
			})

			It("Test d2", func() {
				time.Sleep(1 * time.Second)
			})
		})

	})

	var _ = Describe("Ordered Node 2", Ordered, func() {
		When("B", func() {
			It("Test 1", func() {
				time.Sleep(1 * time.Second)
			})

			It("Test d2", func() {
				time.Sleep(1 * time.Second)
			})
		})
	})
})
