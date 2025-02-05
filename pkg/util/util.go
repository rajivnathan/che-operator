//
// Copyright (c) 2012-2019 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//
package util

import (
	"crypto/tls"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/client-go/discovery"
	"math/rand"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"strings"
	"time"
)


func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func DoRemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func GeneratePasswd(stringLength int) (passwd string) {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	length := stringLength
	buf := make([]rune, length)
	for i := range buf {
		buf[i] = chars[rand.Intn(len(chars))]
	}
	passwd = string(buf)
	return passwd
}

func DetectOpenShift() (bool, error) {
	tests := IsTestMode()
	if !tests {
		kubeconfig, err := config.GetConfig()
		if err != nil {
			return false, err
		}
		discoveryClient, err := discovery.NewDiscoveryClientForConfig(kubeconfig)
		if err != nil {
			return false, err
		}
		apiList, err := discoveryClient.ServerGroups()
		if err != nil {
			return false, err
		}
		apiGroups := apiList.Groups
		for i := 0; i < len(apiGroups); i++ {
			if apiGroups[i].Name == "route.openshift.io" {
				return true, nil
			}
		}
		return false, nil
	}
	return true, nil
}

func GetValue(key string, defaultValue string) (value string) {

	value = key
	if len(key) < 1 {
		value = defaultValue
	}
	return value
}

func IsTestMode() (isTesting bool) {

	testMode := os.Getenv("MOCK_API")
	if len(testMode) == 0 {
		return false
	}
	return true
}

// GetClusterPublicHostname is a hacky way to get OpenShift API public DNS/IP
// to be used in OpenShift oAuth provider as baseURL
func GetClusterPublicHostname() (hostname string, err error) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{}
	kubeApi := os.Getenv("KUBERNETES_PORT_443_TCP_ADDR")
	url := "https://" + kubeApi + "/.well-known/oauth-authorization-server"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("An error occurred when getting API public hostname: %s", err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("An error occurred when getting API public hostname: %s", err)
		return "", err
	}
	var jsonData map[string]interface{}
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		logrus.Errorf("An error occurred when unmarshalling: %s", err)
		return "", err
	}
	hostname = jsonData["issuer"].(string)
	return hostname, nil
}

func GenerateProxyJavaOpts(proxyURL string, proxyPort string, nonProxyHosts string, proxyUser string, proxyPassword string) (javaOpts string) {

	proxyHost := strings.TrimLeft(proxyURL, "https://")
	proxyUserPassword := ""
	if len(proxyUser) > 1 && len(proxyPassword) > 1 {
		proxyUserPassword =
			" -Dhttp.proxyUser=" + proxyUser + " -Dhttp.proxyPassword=" + proxyPassword +
				" -Dhttps.proxyUser=" + proxyUser + " -Dhttps.proxyPassword=" + proxyPassword
	}
	javaOpts =
		" -Dhttp.proxyHost=" + proxyHost + " -Dhttp.proxyPort=" + proxyPort +
			" -Dhttps.proxyHost=" + proxyHost + " -Dhttps.proxyPort=" + proxyPort +
			" -Dhttp.nonProxyHosts='" + nonProxyHosts + "'" + proxyUserPassword
	return javaOpts
}

func GenerateProxyEnvs(proxyHost string, proxyPort string, nonProxyHosts string, proxyUser string, proxyPassword string) (proxyUrl string, noProxy string) {
	proxyUrl = proxyHost + ":" + proxyPort
	if len(proxyUser) > 1 && len(proxyPassword) > 1 {
		protocol := strings.Split(proxyHost, "://")[0]
		host := strings.Split(proxyHost, "://")[1]
		proxyUrl = protocol + "://" + proxyUser + ":" + proxyPassword + "@" + host + ":" + proxyPort
	}

	noProxy = strings.Replace(nonProxyHosts, "|", ",", -1)

	return proxyUrl, noProxy
}
