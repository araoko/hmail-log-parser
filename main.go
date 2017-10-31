package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	shellwords "github.com/mattn/go-shellwords"
)

func main() {
	//tc := newTimeCounter()
	//st := time.Now()
	// test
	var dFlag = flag.String("d", time.Now().Format("2006-01-02"), "log file date in format yyyy-MM-dd")
	var cFlag = flag.String("c", defConfFilePath, "config file path")
	var iFlag = flag.String("i", "", "Remote IP Address")
	var sFlag = flag.Int("s", -1, "Session ID to Query")
	var hFlag = flag.Int("h", -1, "the hour to Query (0-23")
	var mFlag messageFlag
	//var mFlag = flag.String("m", "", "String to search in message")
	flag.Var(&mFlag, "m", "String to search in message. (can appear more than once)")
	var oFlag = flag.String("o", "", "Output file")
	var serverFlag = flag.Bool("server", false, "Only messages from the SMTPD (Server) Service")
	var clientFlag = flag.Bool("client", false, "Only messages from the SMTPC (Client) Service")
	var sessionFlag = flag.Bool("session", false, "return entire session")
	var summaryFlag = flag.Bool("summary", false, "also print summary")

	flag.Parse()
	if !validageFlags(cFlag, dFlag, sFlag, hFlag, sessionFlag, serverFlag, clientFlag) {
		log.Fatal("Flags Invalid")
	}
	out := os.Stdout
	var e error
	if len(*oFlag) != 0 {
		out, e = os.OpenFile(*oFlag, os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0644)
		if e != nil {
			log.Fatal("output file Error: ", e)
		}
		defer out.Close()

	}
	conf := loadConfig(*cFlag)
	conf.confPath = *cFlag
	logFile := getLogFile(*dFlag, conf.LogDir, conf.LogRepoDir)
	if logFile == "" {
		log.Fatal("logfile Not Found")
	}
	var file *os.File
	var err error
	if file, err = os.Open(logFile); err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	logRecords = make([]*logRecord, 0)
	ipIndex := make(map[string][]int)
	sessionIndex := make(map[int][]int)
	dayIndex := make(map[string][]int)
	hourIndex := make(map[string][]int)
	scanner := bufio.NewScanner(file)
	//tc.start()
	for scanner.Scan() {
		r := getLogRecord(scanner.Text())
		if r == nil {
			continue
		}
		logRecords = append(logRecords, r)
		i := len(logRecords) - 1
		ipIndex[r.remoteIP] = updateSlice(ipIndex[r.remoteIP], i)
		sessionIndex[r.sessionID] = updateSlice(sessionIndex[r.sessionID], i)
		dayIndex[r.dateTime.Format("2006-01-02")] = updateSlice(dayIndex[r.dateTime.Format("2006-01-02")], i)
		hourIndex[r.dateTime.Format("2006-01-02_15")] = updateSlice(hourIndex[r.dateTime.Format("2006-01-02_15")], i)
	}
	//scantimeT := tc.stop()
	// check for errors
	if err = scanner.Err(); err != nil {
		log.Fatal(err)
	}

	var hourIndexKey string
	if *hFlag != -1 {
		hourIndexKey = *dFlag + "_" + strconv.FormatInt(int64(*hFlag), 10)
	}
	hourSlice := getSliceFromMapString(hourIndex, hourIndexKey)
	ipSlice := getSliceFromMapString(ipIndex, *iFlag)
	sessionSlice := getSliceFromMapInt(sessionIndex, *sFlag)

	filtered := getCommon(hourSlice, ipSlice, sessionSlice)
	if filtered == nil {
		filtered = getAllIndices()
	}
	s := ""
	if *serverFlag {
		s = "SMTPD"
	}
	if *clientFlag {
		s = "SMTPC"
	}

	if len(s) != 0 {
		filtered = filterSourceIndex(filtered, s)
	}

	if len(mFlag) != 0 && !*sessionFlag {

	}
	//tc.start()
	if *sessionFlag {
		filtered = getAllSessionsRecords(sessionIndex, filtered)
		for i := range mFlag {
			//filtered = getAllSessionsRecords()
			filtered = filterMessageIndex(filtered, mFlag[i:i+1])
			filtered = getAllSessionsRecords(sessionIndex, filtered)
		}
	} else {
		filtered = filterMessageIndex(filtered, mFlag)
	}

	//sessionT := tc.stop()
	fmt.Fprint(out, "Filtered Count: ", len(filtered), "\r\n")

	records := getLogRecordsFromIndices(filtered)
	//tc.start()
	printRecords(out, records)
	//prT := tc.stop()
	//tc.start()
	if *summaryFlag {
		fmt.Fprint(out, "\r\n\r\nSummary for log file: ", logFile, "\r\n")
		printSummary(out, sessionIndex, ipIndex, hourIndex)
	}
	//summaryT := tc.stop()
	//fmt.Println("total time:", time.Now().Sub(st), "\nprint records: ", prT, "\nSession: ", sessionT, "\nScantime: ", scantimeT, "\n Dedup: ", getCommonT, "\nSummary: ", summaryT)
	fmt.Println("Done")
}

func getSliceFromMapInt(m map[int][]int, k int) []int {
	if k == -1 {
		return nil
	}
	if r, ok := m[k]; ok {
		return r
	}
	return make([]int, 0)
}

func convertFlag2HourIndexKey(f int, d string) string {
	if f == -1 {
		return ""
	}
	return d + "_" + strconv.FormatInt(int64(f), 10)
}

func getSliceFromMapString(m map[string][]int, k string) []int {
	if k == "" {
		return nil
	}
	if r, ok := m[k]; ok {
		return r
	}
	return make([]int, 0)
}

func getAllSessionsRecords(sessionMap map[int][]int, recordIndices []int) []int {
	uniqueSIDs := getUniqueSessionIDs(recordIndices)
	result := make([]int, 0)
	for _, sid := range uniqueSIDs {
		result = append(result, sessionMap[sid]...)
	}
	return result
}

func printSummary(out *os.File, sessionIndex map[int][]int, ipIndex, hourIndex map[string][]int) {

	fmt.Fprint(out, "record count: ", len(logRecords), "\r\n")

	fmt.Fprint(out, "IP count: ", len(ipIndex), "\r\n")
	printMapStringKey(out, ipIndex)
	fmt.Fprint(out, "\r\n")

	fmt.Fprint(out, "Session count: ", len(sessionIndex), "\r\n")
	fmt.Fprint(out, "\r\n")

	fmt.Fprint(out, "Hours count: ", len(hourIndex), "\r\n")
	printMapStringKey(out, hourIndex)
	fmt.Fprint(out, "\r\n")

}

func loadConfig(p string) config {
	jsonBytes, err := ioutil.ReadFile(p)
	if err != nil {
		log.Fatal(err)
	}
	var c config
	err = json.Unmarshal(jsonBytes, &c)
	return c
}

func validageFlags(c *string, d *string, s *int, h *int, session *bool, server *bool, client *bool) bool {
	if *server && *client {
		log.Println("Only one of the -server and -client parameters can be used")
		return false
	}
	if *session && *s != -1 {
		log.Println("Only one of the -s and -session flags can be used")
		return false
	}

	if *h > 23 || *h < -1 {
		log.Println("Ivalid value for -h: use a number between 0 and 23")
		return false
	}
	_, err := time.Parse("2006-01-02", *d)
	if err != nil {
		log.Println("invalid -d argument: ", err)
		return false
	}

	fileInfo, err := os.Stat(*c)
	if err == nil && fileInfo.IsDir() {
		log.Println("invalid -c argument: ", *c, "is a directory, it should be a config file")
		return false
	}
	if err != nil && os.IsNotExist(err) {
		log.Println("invalid -c argument: cannot find ", *c, "in the file system")
		return false
	}
	return true
}

func getLogFile(date string, dirs ...string) string {
	filename := "hmailserver_" + date + ".log"
	//"hmailserver_2006-01-02.log"
	for _, dir := range dirs {
		path := filepath.Join(dir, filename)
		m, err := filepath.Glob(path)
		if err != nil {
			log.Fatal(err)
		}
		if len(m) == 0 {
			continue
		}
		return m[0]
	}
	return ""
}

func parseLine(l string) []string {
	args, err := shellwords.Parse(l)
	if err != nil {
		log.Println(err)
		return nil
	}
	return args
}

func getLogRecord(l string) *logRecord {
	ps := parseLine(l)
	if len(ps) != 6 {
		return nil
	}

	tID := string2Int(ps[1])

	if tID == -1 {
		return nil
	}

	sID := string2Int(ps[2])

	if sID == -1 {
		return nil
	}

	t, err := time.Parse(logTimestampLayout, ps[3])
	if err != nil {
		return nil
	}
	m := strings.Split(ps[5], ":")
	txt := strings.Join(m[1:], ":")
	msg := logMessage{
		isSent: m[0] == "SENT",
		text:   txt,
	}
	return &logRecord{ps[0], tID, sID, t, ps[4], msg}
}

func string2Int(s string) int {
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return -1
	}
	return int(i)
}

func updateSlice(v []int, i int) []int {
	if v == nil {
		v = make([]int, 0)
	}
	return append(v, i)
}

func getCommon(a ...[]int) []int {
	s := make([][]int, 0)

	// take care of nil and empty slices first
	for _, i := range a {
		if i == nil {
			continue
		}
		if len(i) == 0 {
			return []int{}
		}
		s = append(s, i)
	}

	if len(s) == 1 {
		return s[0]
	}

	r := s[0]

	for slice := 1; slice < len(s); slice++ {
		r = gc(r, s[slice])
		if len(r) == 0 {
			break
		}
	}
	sort.Ints(r)
	return r
}

func gc(a []int, b []int) []int {
	m := make(map[int]struct{})
	for _, i := range a {
		m[i] = struct{}{}
	}
	r := make([]int, 0)
	for _, j := range b {
		if _, ok := m[j]; ok {
			r = append(r, j)
		}
	}
	return r
}

func filterSourceIndex(a []int, s string) []int {
	r := make([]int, 0)
	for _, i := range a {
		if logRecords[i].source == s {
			r = append(r, i)
		}
	}
	return r
}

func filterMessageIndex(a []int, m messageFlag) []int {
	if m == nil {
		return a
	}
	r := make([]int, 0)
	for _, i := range a {
		matched := true
		for _, j := range m {
			if !strings.Contains(logRecords[i].message.text, j) {
				matched = false
				break
			}

		}
		if matched {
			r = append(r, i)
		}

	}
	return r
}

func getUniqueSessionIDs(a []int) []int {
	r := make([]int, 0)
	m := make(map[int]struct{})
	for _, i := range a {
		sID := logRecords[i].sessionID
		if _, ok := m[sID]; !ok {
			m[sID] = struct{}{}
			r = append(r, sID)
		}
	}
	return r
}
func getAllIndices() []int {
	r := make([]int, len(logRecords))
	for index := 0; index < len(logRecords); index++ {
		r[index] = index
	}
	return r
}

func getLogRecordsFromIndices(indices []int) []*logRecord {
	r := make([]*logRecord, 0)
	for _, i := range indices {
		r = append(r, logRecords[i])
	}
	return r
}

func printRecords(o *os.File, r []*logRecord) {
	fmt.Fprint(o, "Source", "\t", "ThreadID", "\t", "SessionID", "\t", "Timestamp", "\t", "\t", "\t", "RemoteIP", "\t", "Message", "\t\r\n")
	for _, l := range r {
		SoR := "RECEIVED: "
		if l.message.isSent {
			SoR = "SENT: "
		}

		fmt.Fprint(o, l.source, "\t", l.ThreadID, "\t", "\t", l.sessionID, "\t", l.dateTime.Format(logTimestampLayout), "\t", l.remoteIP, "\t", `"`+SoR+l.message.text+`"`, "\r\n")
	}
}

func printMapStringKey(o *os.File, m map[string][]int) {
	sorter := newStringListSorter()

	for k := range m {
		sorter.add(k)
	}
	sortList := sorter.getList()
	for _, k := range sortList {
		v := m[k]
		fmt.Fprint(o, k, " : ", len(v), "\r\n")
	}
}

type stopWatch struct {
	m map[int]time.Time
	i int
}

func newTimeCounter() *stopWatch {
	return &stopWatch{make(map[int]time.Time), 0}
}

func (r *stopWatch) start() int {
	r.i++
	r.m[r.i] = time.Now()
	return r.i
}

func (r *stopWatch) stop(i int) (time.Duration, error) {
	t2 := time.Now()
	if t, ok := r.m[i]; ok {
		delete(r.m, i)
		return t2.Sub(t), nil
	}
	return time.Duration(-1), fmt.Errorf("Instance %d Does not exist In this stopWatch", i)

}

type stringListSorter struct {
	b map[string]struct{}
	s bool
	l []string
}

func newStringListSorter() *stringListSorter {
	return &stringListSorter{make(map[string]struct{}), true, nil}
}

func (n *stringListSorter) add(st string) {
	n.b[st] = struct{}{}
	n.s = false
}

func (n *stringListSorter) sort() {
	n.l = make([]string, 0, len(n.b))
	for k := range n.b {
		n.l = append(n.l, k)
	}
	sort.Strings(n.l)
	n.s = true
}

func (n *stringListSorter) getList() []string {
	if n.s != true {
		n.sort()
	}
	return n.l
}

type messageFlag []string

func (m *messageFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *messageFlag) Set(s string) error {
	*m = append(*m, s)
	return nil
}

type config struct {
	confPath   string
	LogDir     string
	LogRepoDir string
}

type logRecord struct {
	source    string
	ThreadID  int
	sessionID int
	dateTime  time.Time
	remoteIP  string
	message   logMessage
}

type logMessage struct {
	isSent bool
	text   string
}

var logRecords []*logRecord

const (
	logTimestampLayout      = "2006-01-02 15:04:05.000"
	logFile2TimestampLayout = "hmailserver_2006-01-02.log"
	defConfFilePath         = "conf"
)
