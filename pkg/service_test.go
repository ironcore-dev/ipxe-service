package pkg

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
	"os"
)

var _ = Describe("iPXE service without test data", func() {
	Context("Default", func() {
		It("Chain ", func() {
			req, err := http.NewRequest("GET", "/ipxe", nil)
			req.Header.Set("X-FORWARDED-FOR", validIP1)
			Expect(err).ToNot(HaveOccurred())

			rr := httptest.NewRecorder()
			rtr := ipxe.getRouter()
			handler := http.Handler(rtr)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).Should(BeNumerically("==", http.StatusOK))

			expected, err := os.ReadFile("../config/samples/ipxe-default-cm/ipxe")
			Expect(err).ToNot(HaveOccurred())

			By("Expect successful iPXE default response")
			Expect(rr.Body.String()).Should(BeIdenticalTo(string(expected)))
		})

		It("Chain with bad uuid", func() {
			req, err := http.NewRequest("GET", fmt.Sprintf("/ipxe/%s/boot", badUUID), nil)
			req.Header.Set("X-FORWARDED-FOR", validIP1)
			Expect(err).ToNot(HaveOccurred())

			rr := httptest.NewRecorder()
			rtr := ipxe.getRouter()
			handler := http.Handler(rtr)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).Should(BeNumerically("==", http.StatusInternalServerError))
		})

		It("Ignition with bad uuid", func() {
			req, err := http.NewRequest("GET", fmt.Sprintf("/ignition/%s/default", badUUID), nil)
			req.Header.Set("X-FORWARDED-FOR", validIP1)
			Expect(err).ToNot(HaveOccurred())

			rr := httptest.NewRecorder()
			rtr := ipxe.getRouter()
			handler := http.Handler(rtr)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).Should(BeNumerically("==", http.StatusInternalServerError))
		})
	})
})

var _ = Describe("iPXE service with test data", func() {
	Context("Access", func() {
		ctx := context.Background()
		SetupTestData(ctx)

		It("Ignition with valid ip and bad uuid", func() {
			req, err := http.NewRequest("GET", fmt.Sprintf("/ignition/%s/default", badUUID), nil)
			req.Header.Set("X-FORWARDED-FOR", validIP1)
			Expect(err).ToNot(HaveOccurred())

			rr := httptest.NewRecorder()
			rtr := ipxe.getRouter()
			handler := http.Handler(rtr)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).Should(BeNumerically("==", http.StatusInternalServerError))
		})

		It("Ignition with valid ip and uuid", func() {
			req, err := http.NewRequest("GET", fmt.Sprintf("/ignition/%s/default", uuid), nil)
			req.Header.Set("X-FORWARDED-FOR", validIP1)
			Expect(err).ToNot(HaveOccurred())

			rr := httptest.NewRecorder()
			rtr := ipxe.getRouter()
			handler := http.Handler(rtr)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).Should(BeNumerically("==", http.StatusOK))

			expected, err := os.ReadFile("../config/samples/ignition/f2175eb4-e203-11ec-b5d5-3a68dd76b473.ign")
			Expect(err).ToNot(HaveOccurred())

			By("Expect successful iPXE response")
			Expect(rr.Body.String()).Should(BeIdenticalTo(string(expected)))
		})

		It("Ignition with valid ip and empty inventory uuid", func() {
			req, err := http.NewRequest("GET", fmt.Sprintf("/ignition/%s/default", emptyInventoryUUID), nil)
			req.Header.Set("X-FORWARDED-FOR", validIP2)
			Expect(err).ToNot(HaveOccurred())

			rr := httptest.NewRecorder()
			rtr := ipxe.getRouter()
			handler := http.Handler(rtr)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).Should(BeNumerically("==", http.StatusOK))

			expected, err := os.ReadFile("../config/samples/ignition/94925a7e-d7e8-11ec-9bb5-3a68dd71f463.ign")
			Expect(err).ToNot(HaveOccurred())

			By("Expect successful iPXE response")
			Expect(rr.Body.String()).Should(BeIdenticalTo(string(expected)))
		})

		It("Chain with valid emtpy inventory uuid", func() {
			req, err := http.NewRequest("GET", fmt.Sprintf("/ipxe/%s/boot", emptyInventoryUUID), nil)
			req.Header.Set("X-FORWARDED-FOR", validIP1)
			Expect(err).ToNot(HaveOccurred())

			rr := httptest.NewRecorder()
			rtr := ipxe.getRouter()
			handler := http.Handler(rtr)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).Should(BeNumerically("==", http.StatusOK))

			expected, err := os.ReadFile("../config/samples/ipxe-default-cm/boot")
			Expect(err).ToNot(HaveOccurred())

			By("Expect successful iPXE response")
			Expect(rr.Body.String()).Should(BeIdenticalTo(string(expected)))
		})

		It("Chain with valid uuid", func() {
			req, err := http.NewRequest("GET", fmt.Sprintf("/ipxe/%s/boot", uuid), nil)
			req.Header.Set("X-FORWARDED-FOR", validIP1)
			Expect(err).ToNot(HaveOccurred())

			rr := httptest.NewRecorder()
			rtr := ipxe.getRouter()
			handler := http.Handler(rtr)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).Should(BeNumerically("==", http.StatusOK))

			expected, err := os.ReadFile("../config/samples/configmap/ipxe-f2175eb4-e203-11ec-b5d5-3a68dd76b473")
			Expect(err).ToNot(HaveOccurred())

			By("Expect successful iPXE response")
			Expect(rr.Body.String()).Should(BeIdenticalTo(string(expected)))
		})

		It("Chain with bad ip and valid uuid", func() {
			req, err := http.NewRequest("GET", fmt.Sprintf("/ipxe/%s/boot", uuid), nil)
			req.Header.Set("X-FORWARDED-FOR", badIP)
			Expect(err).ToNot(HaveOccurred())

			rr := httptest.NewRecorder()
			rtr := ipxe.getRouter()
			handler := http.Handler(rtr)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).Should(BeNumerically("==", http.StatusInternalServerError))
		})
	})
})
