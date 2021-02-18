package lib

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb"
	"go.uber.org/ratelimit"
	"golang.org/x/net/publicsuffix"
)

var (
	kill = &Kill{shotsCount: 0}
	rl   ratelimit.Limiter
)

// Kill a collection of properties for a set of hits
type Kill struct {
	shotsCount    int
	GunsCount     int           `yaml:"concurrency"`
	AttemptsCount int           `yaml:"loopcount"`
	Timeout       time.Duration `yaml:"timeout"`
	Rate          int           `yaml:"ratepersecond"`
	gun           *Gun
	victim        *Victim
}

// GetKill get collection of properties for a set of hits
func GetKill() *Kill {
	return kill
}

// SetGun set target for a call
func (k *Kill) SetGun(gun *Gun) {
	k.gun = gun
}

// SetVictim set target for hits
func (k *Kill) SetVictim(victim *Victim) {
	k.victim = victim
}

// Prepare get ready to hit targets
func (k *Kill) Prepare() error {
	reporter.ln()
	reporter.log("prepare kill")

	err := k.victim.prepare()
	k.gun.prepare()

	if k.GunsCount == 0 {
		k.GunsCount = 1
	}
	reporter.log("guns count - %v", k.GunsCount)

	if k.AttemptsCount == 0 {
		k.AttemptsCount = 1
	}
	reporter.log("attempts count - %v", k.AttemptsCount)

	if k.Timeout == 0 {
		k.Timeout = 2
	}
	if k.Rate == 0 {
		k.Rate = 1000
	}
	reporter.log("timeout - %v", k.GunsCount)
	reporter.log("shots count - %v", k.shotsCount)

	return err
}

// Start begin a set of hits
func (k *Kill) Start() {
	rate := k.Rate

	rl = ratelimit.New(rate, ratelimit.WithoutSlack)
	if k.Rate == 1000 {
		rl = ratelimit.NewUnlimited()
	}
	// fmt.Println("Rate", rate)

	reporter.ln()
	reporter.log("start kill")

	// отдаем рутинам все ядра процессора
	runtime.GOMAXPROCS(runtime.NumCPU())
	// считаем кол-во результатов
	hitsCount := k.GunsCount * k.AttemptsCount * k.shotsCount
	reporter.log("hits count: %v", hitsCount)
	hitsByAttempt := hitsCount / k.AttemptsCount
	reporter.log("hits by attempt: %v", hitsByAttempt)

	// создаем програсс бар
	bar := pb.StartNew(hitsCount)
	group := new(sync.WaitGroup)
	// создаем канал результатов
	hits := make(chan *Hit, hitsCount)
	shots := make(chan *Shot, hitsCount)
	// запускаем повторения заданий,
	// если в настройках не указано кол-во повторений,
	// тогда программа сделает одно повторение
	for i := 0; i < k.AttemptsCount; i++ {
		reporter.log("attempt - %v", i)
		group.Add(hitsByAttempt)
		// запускаем конкуретные задания,
		// если в настройках не указано кол-во заданий,
		// тогда программа сделает одно задание
		for j := 0; j < k.GunsCount; j++ {
			go func() {
				// Get new rate limit token
				killer := new(Killer)
				killer.setVictim(k.victim)
				killer.setGun(k.gun)

				go killer.fire(hits, shots, group, bar)
				reporter.log("killer - %v charge", j)
				go killer.charge(shots)
			}()
		}
		group.Wait()
	}

	close(shots)
	close(hits)
	// аггрегируем результаты задания и выводим статистику в консоль
	reporter.report(k, hits)
}

// Shot definition of properties required for a call to a target
type Shot struct {
	cartridge *Cartridge
	request   *http.Request
	client    *http.Client
	transport *http.Transport
}

// Killer definition of
type Killer struct {
	victim  *Victim
	gun     *Gun
	session *Caliber
}

func (k *Killer) setVictim(victim *Victim) {
	k.victim = victim
}

func (k *Killer) setGun(gun *Gun) {
	k.gun = gun
}

func (k *Killer) charge(shots chan *Shot) {

	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		reporter.log("cookie don't created - %v", err)
	}
	client := new(http.Client)
	client.Jar = jar
	k.chargeCartidges(shots, client, k.gun.Cartridges)
}

func (k *Killer) chargeCartidges(shots chan<- *Shot, client *http.Client, cartridges Cartridges) {

	for _, cartridge := range cartridges {
		if cartridge.getMethod() == RANDOM_METHOD || cartridge.getMethod() == SYNC_METHOD {
			k.chargeCartidges(shots, client, cartridge.getChildren())
		} else {
			isPostRequest := cartridge.getMethod() == POST_METHOD
			var timeout time.Duration
			if cartridge.timeout > 0 {
				timeout = cartridge.timeout
			} else {
				timeout = kill.Timeout
			}

			shot := new(Shot)
			shot.cartridge = cartridge
			shot.client = client
			shot.transport = &http.Transport{
				Dial: func(network, addr string) (conn net.Conn, err error) {
					return net.DialTimeout(network, addr, time.Second*timeout)
				},
				ResponseHeaderTimeout: time.Second * timeout,
				DisableKeepAlives:     true,
			}

			reqURL := new(url.URL)
			reqURL.Scheme = k.victim.Scheme
			reqURL.Host = k.victim.Host

			pathParts := strings.Split(cartridge.getPathAsString(k), "?")
			reqURL.Path = pathParts[0]
			if len(pathParts) == 2 {
				val, _ := url.ParseQuery(pathParts[1])
				reqURL.RawQuery = val.Encode()
			} else {
				reqURL.RawQuery = ""
			}

			var body bytes.Buffer
			request, err := http.NewRequest(cartridge.getMethod(), reqURL.String(), &body)
			if err == nil {
				k.setFeatures(request, k.gun.Features)
				k.setFeatures(request, cartridge.bulletFeatures)
				if isPostRequest {

					switch request.Header.Get("Content-Type") {
					case "multipart/form-data":
						writer := multipart.NewWriter(&body)
						for _, feature := range cartridge.chargeFeatures {
							writer.WriteField(feature.name, feature.String(k))
						}
						writer.Close()
						request.Body = ioutil.NopCloser(bytes.NewReader(body.Bytes()))
						request.Header.Set("Content-Type", writer.FormDataContentType())
						break
					case "application/json":
						for _, feature := range cartridge.chargeFeatures {
							if feature.name == "raw_body" {
								body.WriteString(feature.String(k))
							}
						}
						request.Body = ioutil.NopCloser(bytes.NewReader(body.Bytes()))
						request.Header.Set("Content-Type", "application/json")
						break
					default:
						params := url.Values{}
						for _, feature := range cartridge.chargeFeatures {
							params.Set(feature.name, feature.String(k))
						}
						body.WriteString(params.Encode())
						request.Body = ioutil.NopCloser(bytes.NewReader(body.Bytes()))
						if len(request.Header.Get("Content-Type")) == 0 {
							request.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
						}
						break

					}
					request.ContentLength = int64(body.Len())
				}

				if reporter.Debug {
					reporter.log("create request:")
					dump, _ := httputil.DumpRequest(request, true)
					reporter.log(string(dump))
				}
				shot.request = request
				shots <- shot
			} else {
				reporter.log("request don't created, error: %v", err)
			}
		}
	}
}

func (k *Killer) setFeatures(request *http.Request, features Features) {
	for _, feature := range features {
		request.Header.Set(feature.name, feature.String(k))
	}
}

func (k *Killer) fire(hits chan<- *Hit, shots <-chan *Shot, group *sync.WaitGroup, bar *pb.ProgressBar) {
	for shot := range shots {
		rl.Take()

		hit := new(Hit)
		hit.shot = shot
		shot.client.Transport = shot.transport
		hit.startTime = time.Now()
		resp, err := shot.client.Do(shot.request)
		hit.endTime = time.Now()
		bar.Increment()
		if err == nil {
			if reporter.Debug {
				dump, _ := httputil.DumpResponse(resp, true)
				reporter.log(string(dump))
			}
			hit.response = resp
			hit.responseBody, _ = ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
		} else {
			reporter.log("response don't received, error: %v", err)
		}
		hits <- hit
		group.Done()
	}
}

type Hit struct {
	startTime    time.Time
	endTime      time.Time
	shot         *Shot
	response     *http.Response
	responseBody []byte
}

const (
	HTTP_SCHEME  = "http"
	HTTPS_SCHEME = "https"
)

type Victim struct {
	Scheme string `yaml:"scheme"`
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
}

func NewVictim() *Victim {
	return new(Victim)
}

func (v *Victim) prepare() error {
	if len(v.Scheme) > 0 && (v.Scheme != HTTP_SCHEME && v.Scheme != HTTPS_SCHEME) {
		return errors.New("invalid scheme")
	}

	if len(v.Host) == 0 {
		return errors.New("invalid host")
	}

	if len(v.Scheme) == 0 {
		v.Scheme = HTTP_SCHEME
	}
	reporter.log("scheme - %v", v.Scheme)

	if v.Port == 0 {
		v.Port = 80
	}
	reporter.log("port - %v", v.Port)

	if v.Port != 80 {
		v.Host = fmt.Sprintf("%s:%d", v.Host, v.Port)
	}
	reporter.log("host - %v", v.Host)

	return nil
}
