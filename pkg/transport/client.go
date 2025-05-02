package transport

import (
	"fmt"
	"time"
	"sync"

	//"drill/pkg/obfuscate"
	"drill/pkg/proxy"
)

type TransportClient struct {
	HttpsProxyAddress string	
	RemoteServerAddress string
}

func NewTransportClient(
	https_proxy_host string,
	https_proxy_port uint16, 
	remote_host string, 
	remote_port uint16,
) TransportClient {
	https_proxy_addr := fmt.Sprintf("%s:%v", https_proxy_host, https_proxy_port)	
	remote_addr := fmt.Sprintf("%s:%v", remote_host, remote_port)	

	return TransportClient {
		https_proxy_addr,
		remote_addr,
	}
}

func (txp *TransportClient) Run() {
	fmt.Println("Run Transport Client")
	go RunHttpsProxy(txp.HttpsProxyAddress)	

	var wg sync.WaitGroup
	wg.Add(1)
	go RunTransportClient(txp.RemoteServerAddress, &wg)
	wg.Wait()
}

func RunHttpsProxy(addr string) {
	https_proxy := proxy.NewHttpsProxy(addr)
	https_proxy.Run()
}

func RunTransportClient(addr string, wg *sync.WaitGroup) {
	fmt.Println("RunTransportclient")

	for {
		time.Sleep(5 * time.Second)
		fmt.Println("Done")
		break
	}

	wg.Done()
}