package apisix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/eavesmy/golang-lib/net"
)

/*
curl http://127.0.0.1:9180/apisix/admin/routes/apisix.eva7base.com -H 'X-API-KEY: edd1c9f034335f136f87ad84b625c8f1' -X PUT -i -d '
{
    "uri": "/*",
    "hosts": ["apisix.eva7base.com"],
    "remote_addrs": ["0.0.0.0/0"],
    "methods": ["PUT", "GET","POST","PATCH","DELETE","OPTIONS","HEAD","CONNECT","TRACE"],
    "enable_websocket": false,
    "upstream": {
        "type": "roundrobin",
        "nodes": {
            "127.0.0.1:9000": 1
        }
    }
}'
*/

/*
$ curl http://127.0.0.1:9080/apisix/admin/services/201  -H 'X-API-KEY: edd1c9f034335f136f87ad84b625c8f1' -X PUT -i -d '
{
    "plugins": {
        "limit-count": {
            "count": 2,
            "time_window": 60,
            "rejected_code": 503,
            "key": "remote_addr"
        }
    },
    "enable_websocket": true,
    "upstream": {
        "type": "roundrobin",
        "nodes": {
            "39.97.63.215:80": 1
        }
    }
}'
*/

func (s *Svc) registerService() error {

	uri := "/apisix/admin/services/"
	resp, err := s.Get(uri + s.Name)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	inetip := s.InetIp

	if inetip == "" {
		inetip = net.GetIntranetIp()[0]
		s.InetIp = inetip
	}

	inetip += ":" + s.Port

	if s.Plugins == nil {
		s.Plugins = map[string]interface{}{}
	}

	fmt.Println(uri+s.Name, resp.StatusCode, 12333)

	fmt.Println("support websocket?", s.EnableWebsocket)

	if resp.StatusCode >= 404 {

		upstream := Upstream{
			Type:  "roundrobin",
			Nodes: map[string]int{inetip: 1}, // 权重做自增变量
			Checks: &Checks{
				Active: &Active{
					Timeout:  1,
					HTTPPath: "/" + s.Name + "/healthCheck",
					Host:     s.Hosts[0],
					Healthy: &Healthy{
						HTTPStatuses: []int{200},
						Successes:    3,
					},
					Unhealthy: &Unhealthy{
						HTTPStatuses: []int{502, 503, 504},
						HTTPFailures: 2,
						TCPFailures:  3,
					},
					ReqHeaders: []string{"User-Agent: curl/7.29.0"},
				},
				Passive: &Passive{
					Healthy: &Healthy{
						HTTPStatuses: []int{200},
						Successes:    3,
					},
					Unhealthy: &Unhealthy{
						HTTPStatuses: []int{502, 503, 504},
						HTTPFailures: 2,
						TCPFailures:  3,
					},
				},
			},
		}

		if s.HTTP2 {
			upstream.Scheme = "grpc"
		} else {
			upstream.Scheme = "http"
		}

		resp, err = s.Put(uri+s.Name, encode(&Service{
			Name:             s.Name,
			Plugins:          s.Plugins,
			Enable_Websocket: s.EnableWebsocket,
			Upstream:         upstream},
		))

		defer resp.Body.Close()
	} else {

		upstream := Upstream{
			Nodes: map[string]int{inetip: 2},
			Type:  "roundrobin",
			Checks: &Checks{
				Active: &Active{
					Timeout:  1,
					HTTPPath: "/" + s.Name + "/healthCheck",
					Host:     s.Hosts[0],
					Healthy: &Healthy{
						HTTPStatuses: []int{200},
						Successes:    3,
					},
					Unhealthy: &Unhealthy{
						HTTPStatuses: []int{502, 503, 504},
						HTTPFailures: 2,
						TCPFailures:  3,
					},
					ReqHeaders: []string{"User-Agent: curl/7.29.0"},
				},
				Passive: &Passive{
					Healthy: &Healthy{
						HTTPStatuses: []int{200},
						Successes:    3,
					},
					Unhealthy: &Unhealthy{
						HTTPStatuses: []int{502, 503, 504},
						HTTPFailures: 2,
						TCPFailures:  3,
					},
				},
			},
		}
		if s.HTTP2 {
			upstream.Scheme = "grpc"
		} else {
			upstream.Scheme = "http"
		}

		resp, err = s.Put(uri+s.Name, encode(&Service{
			Name:             s.Name,
			Plugins:          s.Plugins,
			Enable_Websocket: s.EnableWebsocket,
			Upstream:         upstream},
		))

		defer resp.Body.Close()
	}

	fmt.Println("Registe service [", s.Name, "] status code:", resp.StatusCode)

	a, _ := ioutil.ReadAll(resp.Body)

	fmt.Println(string(a))

	return nil
}

func (s *Svc) registerRouter(router string, ttls ...time.Duration) error {

	uri := "/apisix/admin/routes/" + s.Name

	if len(ttls) < 0 {
		uri += "?ttl=" + fmt.Sprintf("%d", int(ttls[0].Seconds()))
	}

	resp, err := s.Get(uri)

	if err != nil {
		fmt.Println(err)
		return err
	}

	defer resp.Body.Close()
	inetip := s.InetIp
	if inetip == "" {
		inetip = net.GetIntranetIp()[0]
		s.InetIp = inetip
	}
	inetip += ":" + s.Port

	if s.Hosts == nil {
		s.Hosts = []string{"127.0.0.1"}
	}

	if s.Remote_Addrs == nil {
		s.Remote_Addrs = []string{"0.0.0.0/0"}
	}
	if s.Methods == nil {
		s.Methods = []string{"PUT", "GET", "POST", "PATCH", "DELETE", "OPTIONS", "HEAD", "CONNECT", "TRACE"}
	}

	if resp.StatusCode >= 200 {
		resp, err = s.Put(uri, encode(&Router{
			Uri:              router,
			Hosts:            s.Hosts,
			Remote_Addrs:     s.Remote_Addrs,
			Methods:          s.Methods,
			Enable_Websocket: s.EnableWebsocket,
			Service_Id:       s.Name,
			Name:             s.Name,
			// Version:          s.Version,
		}))

		if resp.Body != nil {
			b, _ := ioutil.ReadAll(resp.Body)
			fmt.Println(string(b), 12333)
		}
		if err != nil {
			fmt.Println(err)
			return err
		}
	}

	defer resp.Body.Close()

	fmt.Println("Registe router [", router, "] status code:", resp.StatusCode)
	// b, _ := ioutil.ReadAll(resp.Body)
	// fmt.Println(string(b))

	return nil
}

func encode(d interface{}) io.Reader {
	b, _ := json.Marshal(d)

	return bytes.NewBuffer(b)
}
