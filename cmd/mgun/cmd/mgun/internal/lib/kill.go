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

	"math/rand"

	"github.com/cheggaaa/pb"
	"go.uber.org/ratelimit"
	"golang.org/x/net/publicsuffix"
)

var (
	kill            = &Attack{shotsCount: 0}
	rl              ratelimit.Limiter
	randomDelayMsec int = 0
)

// Attack a collection of properties for a set of hits
type Attack struct {
	shotsCount          int
	CallCollectionCount int           `yaml:"concurrency"`
	AttemptsCount       int           `yaml:"loopcount"`
	Timeout             time.Duration `yaml:"timeout"`
	Rate                int           `yaml:"ratepersecond"`
	RandomDelayMs       int           `yaml:"randomdelayms"`
	callCollection      *CallCollection
	target              *Target
}

// GetAttack get collection of properties for a set of hits
func GetAttack() *Attack {
	return kill
}

// SetGun set target for a call
func (a *Attack) SetGun(callCollection *CallCollection) {
	a.callCollection = callCollection
}

// SetTarget set target for hits
func (a *Attack) SetTarget(target *Target) {
	a.target = target
}

// Prepare get ready to hit targets
func (a *Attack) Prepare() error {
	reporter.ln()
	reporter.log("prepare kill")

	err := a.target.prepare()
	a.callCollection.prepare()

	if a.CallCollectionCount == 0 {
		a.CallCollectionCount = 1
	}
	reporter.log("callcollection count - %v", a.CallCollectionCount)

	if a.AttemptsCount == 0 {
		a.AttemptsCount = 1
	}
	reporter.log("attempts count - %v", a.AttemptsCount)

	if a.Timeout == 0 {
		a.Timeout = 2
	}
	if a.Rate == 0 {
		a.Rate = 1000
	}
	if a.RandomDelayMs > 0 {
		randomDelayMsec = a.RandomDelayMs
	}

	reporter.log("timeout - %v", a.CallCollectionCount)
	reporter.log("shots count - %v", a.shotsCount)

	return err
}

// Start begin a set of hits
func (a *Attack) Start() {
	rate := a.Rate

	rl = ratelimit.New(rate, ratelimit.WithoutSlack)
	if a.Rate == 1000 {
		rl = ratelimit.NewUnlimited()
	}
	// fmt.Println("Rate", rate)

	reporter.ln()
	reporter.log("start kill")

	// отдаем рутинам все ядра процессора
	runtime.GOMAXPROCS(runtime.NumCPU())
	// считаем кол-во результатов
	hitsCount := a.CallCollectionCount * a.AttemptsCount * a.shotsCount
	reporter.log("hits count: %v", hitsCount)
	hitsByAttempt := hitsCount / a.AttemptsCount
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
	for i := 0; i < a.AttemptsCount; i++ {
		reporter.log("attempt - %v", i)
		group.Add(hitsByAttempt)
		// запускаем конкуретные задания,
		// если в настройках не указано кол-во заданий,
		// тогда программа сделает одно задание
		for j := 0; j < a.CallCollectionCount; j++ {
			go func() {
				// Get new rate limit token
				killer := new(Killer)
				killer.setTarget(a.target)
				killer.setGun(a.callCollection)

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
	reporter.report(a, hits)
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
	target         *Target
	callCollection *CallCollection
	session        *Caliber
}

func (k *Killer) setTarget(target *Target) {
	k.target = target
}

func (k *Killer) setGun(callCollection *CallCollection) {
	k.callCollection = callCollection
}

func (k *Killer) charge(shots chan *Shot) {

	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		reporter.log("cookie wasn't created - %v", err)
	}
	client := new(http.Client)
	client.Jar = jar
	k.chargeCartidges(shots, client, k.callCollection.Cartridges)
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
			reqURL.Scheme = k.target.Scheme
			reqURL.Host = k.target.Host

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
				k.setFeatures(request, k.callCollection.Features)
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
				reporter.log("request not created, error: %v", err)
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

		// Delay for a random number of milliseconds if configured to
		if randomDelayMsec > 0 {
			rand.Seed(time.Now().UnixNano())
			n := rand.Intn(randomDelayMsec) // n will be between 0 and the value
			time.Sleep(time.Duration(n) * time.Millisecond)
		}

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

type Target struct {
	Scheme string `yaml:"scheme"`
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
}

func NewTarget() *Target {
	return new(Target)
}

func (v *Target) prepare() error {
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
