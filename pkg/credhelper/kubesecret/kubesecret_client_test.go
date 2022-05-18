/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package kubesecret

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
)

var _ = Describe("Kube Secrets Client", func() {
	Context("CRUD operations for Kubesecret client", func() {
		It(fmt.Sprintf("Should create, retrive, update, and delete secret in k8s"), func() {
			//Test data for secret creation
			secretName := "secret" + strconv.FormatInt(GinkgoRandomSeed(), 10)
			secretNamespace := "default"
			data := map[string][]byte{
				"secret": []byte("test"),
				"data":   []byte("default"),
			}
			labels := map[string]string{
				"lablel_def": "default_label",
			}

			//Test context and k8 client for test case execution
			ctx := context.Background()
			client := &KubeSecretClient{
				Client:  k8sClient,
				Log:     loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("kubesecret_client_test")},
				Metrics: metrics.Init("KubeSecret", loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("kubesecret_client_test")}),
			}

			Context("creating secret with secret client", func() {
				_, err := client.CreateSecret(ctx, secretName, secretNamespace, labels, data)
				Expect(err).To(BeNil())
			})

			Context("ensuring secret exists using k8s client", func() {
				d, err := client.GetSecret(ctx, secretName, secretNamespace)
				Expect(err).To(BeNil())
				for k, v := range d {
					Expect(data[k]).To(Equal(v))
				}
			})

			Context("updating secret with secret client", func() {
				updatedData := map[string][]byte{
					"secret": []byte("Updated_test"),
					"data":   []byte("Updated_default"),
					"value":  []byte("Updated value"),
				}
				_, err := client.UpdateSecret(ctx, secretName, secretNamespace, labels, updatedData)
				Expect(err).To(BeNil())

				d, err := client.GetSecret(ctx, secretName, secretNamespace)
				Expect(err).To(BeNil())
				for k, v := range d {
					Expect(updatedData[k]).To(Equal(v))
				}
			})

			Context("creating secret with same name within same namespace", func() {
				_, err := client.CreateSecret(ctx, secretName, secretNamespace, labels, data)
				Expect(err).ToNot(BeNil())
			})

			Context("delete secret and ensure it is gone", func() {
				_, err := client.DeleteSecret(ctx, secretName, secretNamespace)
				Expect(err).To(BeNil())

				_, err = client.GetSecret(ctx, secretName, secretNamespace)
				Expect(err).ToNot(BeNil())
			})
		})
	})
})
