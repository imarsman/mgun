package lib

import (
	"fmt"
	"io/ioutil"
	"math"
	"strings"
	"time"

	tm "github.com/buger/goterm"
	"github.com/cznic/mathutil"
	hm "github.com/dustin/go-humanize"
	"gitlab.xml.team/imarsman/mgun/cmd/mgun/internal/opt"
)

const (
	// EmptySign empty string
	EmptySign = ""
)

var (
	reporter = new(Reporter)
	output   = ""
)

// SetOutput set the output file path
func SetOutput(val string) {
	output = val
}

// GetReporter get flags for reporting
func GetReporter() *Reporter {
	return reporter
}

// Reporter flags for reporting
type Reporter struct {
	Debug  bool   `yaml:"debug"`
	Output string `yaml:"output"`
}

func (r *Reporter) log(message string, args ...interface{}) {
	if r.Debug {
		message = fmt.Sprintf(message, args...)
		fmt.Println(message)
	}
}

func (r *Reporter) ln() {
	r.log(EmptySign)
}

func (r *Reporter) report(attack *Attack, hits <-chan *Hit) {
	var startTime int64
	var endTime int64
	requestsPerSeconds := make(map[int64]map[int]int)
	reports := make(map[int]*RequestReport)
	hitsTable := tm.NewTable(0, 0, 2, ' ', 0)
	fmt.Fprintf(hitsTable, "#\tRequest\n")
	fmt.Fprintf(hitsTable, "\t%-8s\t%-8s\t%-8s\t%-8s\t%-8s\t%-8s\t%-1s\t%-10s\t%-7s\n", "Compl", "Fail.", "Min/s", "Max/s", "Avg/s.", "Avail%", "Min/Ave/Max req/s. ", "Cont len", "Total trans")
	for hit := range hits {
		if startTime == 0 {
			startTime = hit.startTime.Unix()
		} else {
			startTime = mathutil.MinInt64(startTime, hit.startTime.Unix())
		}
		key := hit.shot.cartridge.id
		if report, ok := reports[key]; ok {
			report.update(hit)
		} else {
			report := NewRequestReport(hit)
			reports[key] = report
		}

		if _, ok := requestsPerSeconds[hit.endTime.Unix()]; ok {
			requestsPerSeconds[hit.endTime.Unix()][hit.shot.cartridge.id]++
		} else {
			requestsPerSeconds[hit.endTime.Unix()] = make(map[int]int)
			requestsPerSeconds[hit.endTime.Unix()][hit.shot.cartridge.id] = 1
		}

		if endTime < 0 {
			endTime = hit.endTime.Unix()
		} else {
			endTime = mathutil.MaxInt64(endTime, hit.endTime.Unix())
		}
	}

	var totalRequests int
	var completeRequests int
	var failedRequests int
	var availability float64
	var totalRequestPerSeconds float64
	var totalTransferred int64

	reportsCount := float64(len(reports))
	cartridges := attack.callCollection.Cartridges.toPlainSlice()
	for _, cartridge := range cartridges {

		if report, ok := reports[cartridge.id]; ok {
			counts := make([]int, 0)
			for _, countByID := range requestsPerSeconds {
				if count, ok := countByID[cartridge.id]; ok {
					counts = append(counts, count)
				}
			}
			var minRequestPerSecond int64
			var avgRequestPerSecond float64
			var maxRequestPerSecond int64
			for _, count := range counts {
				count64 := int64(count)
				if minRequestPerSecond == 0 {
					minRequestPerSecond = count64
				} else {
					minRequestPerSecond = mathutil.MinInt64(minRequestPerSecond, count64)
				}
				avgRequestPerSecond += float64(count)
				if maxRequestPerSecond == 0 {
					maxRequestPerSecond = count64
				} else {
					maxRequestPerSecond = mathutil.MaxInt64(maxRequestPerSecond, count64)
				}
			}
			avgRequestPerSecond = avgRequestPerSecond / float64(len(counts))

			name := r.getRequestName(cartridge)
			totalRequests += report.totalRequests
			completeRequests += report.completeRequests
			failedRequests += report.failedRequests
			availability += report.getAvailability()
			totalTransferred += report.totalTransferred
			totalRequestPerSeconds += avgRequestPerSecond

			fmt.Fprintf(
				hitsTable, "%d.\t%s\n",
				cartridge.id,
				name,
			)

			fmt.Fprintf(
				hitsTable, "\t%-8d\t%-8d\t%-8.3f\t%-8.3f\t%-8.3f\t%-8.2f\t%-2d/ ~ %-2.2f / %-6d\t%-10s\t%-6s\n\n",
				report.completeRequests,
				report.failedRequests,
				report.minTime,
				report.maxTime,
				report.getAvgTime(),
				report.getAvailability(),
				minRequestPerSecond,
				avgRequestPerSecond,
				maxRequestPerSecond,
				hm.Bytes(uint64(report.contentLength)),
				hm.Bytes(uint64(report.totalTransferred)),
			)
		}
	}

	targetTable := tm.NewTable(0, 0, 2, ' ', 0)
	fmt.Fprintf(targetTable, "Server Hostname:\t%s\n", attack.target.Host)
	fmt.Fprintf(targetTable, "Server Port:\t%d\n", attack.target.Port)
	fmt.Fprintf(targetTable, "Concurrency Level:\t%d\n", attack.CallCollectionCount)
	fmt.Fprintf(targetTable, "Rate per second:\t%d\n", attack.Rate)
	fmt.Fprintf(targetTable, "Random delay ms:\t%d\n", attack.RandomDelayMs)
	fmt.Fprintf(targetTable, "Loop count:\t%d\n", attack.AttemptsCount)
	fmt.Fprintf(targetTable, "Timeout:\t%d seconds\n", attack.Timeout)
	fmt.Fprintf(targetTable, "Time taken for tests:\t%d seconds\n", int(time.Unix(endTime, 0).Sub(time.Unix(startTime, 0)).Seconds()))
	fmt.Fprintf(targetTable, "Total requests:\t%d\n", totalRequests)
	fmt.Fprintf(targetTable, "Complete requests:\t%d\n", completeRequests)
	fmt.Fprintf(targetTable, "Failed requests:\t%d\n", failedRequests)
	fmt.Fprintf(targetTable, "Availability:\t%.2f%%\n", availability/reportsCount)
	fmt.Fprintf(targetTable, "Requests per second:\t~ %.2f\n", totalRequestPerSeconds/float64(len(cartridges)))
	fmt.Fprintf(targetTable, "Total transferred:\t%s\n", hm.Bytes(uint64(totalTransferred)))

	fmt.Println(EmptySign)
	fmt.Println(EmptySign)
	fmt.Println(targetTable)
	fmt.Println(hitsTable)

	// Write output if something has been specified in config or as commandline option
	if opt.Output != "" {
		var b strings.Builder
		fmt.Fprintln(&b, targetTable)
		fmt.Fprintln(&b, hitsTable)

		err := ioutil.WriteFile(opt.Output, []byte(b.String()), 0644)
		if err != nil {
			fmt.Printf("Problem writing report to file %s, %v\n", opt.Output, err)
		} else {
			fmt.Printf("Wrote report to file %s\n", opt.Output)
		}
	}
}

func (r *Reporter) getRequestName(cartridge *Cartridge) string {
	return fmt.Sprintf("%s %s", cartridge.getMethod(), cartridge.path.rawDescription)
}

func NewRequestReport(hit *Hit) *RequestReport {
	return new(RequestReport).create(hit)
}

type RequestReport struct {
	totalRequests     int
	startTime         time.Time
	endTime           time.Time
	minTime           float64
	maxTime           float64
	completeRequests  int
	failedRequests    int
	requestsPerSecond float64
	totalTransferred  int64
	totalTime         float64
	contentLength     int64
}

func (this *RequestReport) create(hit *Hit) *RequestReport {
	timeRequest := this.getDiffSeconds(hit)
	this.minTime = timeRequest
	this.maxTime = timeRequest
	this.totalTime = timeRequest
	this.updateTotalRequests()
	this.updateTotalTransferred(hit)
	this.checkResponseStatusCode(hit)
	this.startTime = hit.startTime
	this.endTime = hit.endTime
	return this
}

func (sr *RequestReport) getDiffSeconds(hit *Hit) float64 {
	return hit.endTime.Sub(hit.startTime).Seconds()
}

func (sr *RequestReport) checkResponseStatusCode(hit *Hit) {
	shot := hit.shot
	if hit.shot.request != nil && hit.response != nil {
		statusCode := hit.response.StatusCode
		if sr.inArray(statusCode, shot.cartridge.failedStatusCodes) {
			sr.failedRequests++
		} else if sr.inArray(statusCode, shot.cartridge.successStatusCodes) {
			sr.completeRequests++
		} else {
			sr.failedRequests++
		}
	} else {
		sr.failedRequests++
	}
}

func (sr *RequestReport) inArray(a int, array []int) bool {
	for _, b := range array {
		if a == b {
			return true
		}
	}
	return false
}

func (sr *RequestReport) updateTotalRequests() {
	sr.totalRequests++
}

func (sr *RequestReport) updateTotalTransferred(hit *Hit) {
	if hit.response != nil {
		sr.totalTransferred += int64(len(hit.responseBody))
		if sr.contentLength == 0 {
			sr.contentLength = sr.totalTransferred
		}
	}
}

func (sr *RequestReport) updateRequestsPerSecond(timeRequest float64) {
	if timeRequest == 0 {
		timeRequest = 1
	}
	if sr.requestsPerSecond == 0 {
		sr.requestsPerSecond = 1 / timeRequest
	} else {
		sr.requestsPerSecond = ((1 / timeRequest) + sr.requestsPerSecond) / 2
	}
	reporter.log("time request: %v, requests per second: %v, avg requests per second: %v", timeRequest, 1/timeRequest, sr.requestsPerSecond)
}

func (sr *RequestReport) update(hit *Hit) *RequestReport {
	timeRequest := sr.getDiffSeconds(hit)
	sr.minTime = math.Min(sr.minTime, timeRequest)
	sr.maxTime = math.Max(sr.maxTime, timeRequest)
	sr.totalTime += timeRequest
	sr.updateTotalRequests()
	sr.updateTotalTransferred(hit)
	sr.checkResponseStatusCode(hit)
	sr.endTime = hit.endTime
	return sr
}

func (sr *RequestReport) getAvgTime() float64 {
	return (sr.minTime + sr.maxTime) / 2
}

func (sr *RequestReport) getAvailability() float64 {
	return float64(sr.completeRequests) * 100 / float64(sr.totalRequests)
}
