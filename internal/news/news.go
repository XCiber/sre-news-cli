package news

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/XCiber/sre-news-cli/internal/utils"
	log "github.com/sirupsen/logrus"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

const APIBaseURL = "https://news.b2b.prod.env:8443"

//go:embed slack.tmpl
var slackTemplate string

type Client struct {
	APIBaseURL string
	Holidays   *NasdaqHolidays
	Client     *http.Client
}

type Pagination struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
	Total  int `json:"total"`
}

type Data struct {
	ID                int       `json:"id"`
	MmsID             int       `json:"mms_id"`
	DmID              int       `json:"dm_id"`
	Created           time.Time `json:"created"`
	Updated           time.Time `json:"updated"`
	Title             string    `json:"title"`
	Symbol            string    `json:"symbol"`
	MmsSymbols        string    `json:"mms_symbols"`
	DmSymbols         string    `json:"dm_symbols"`
	Leverage          int       `json:"leverage"`
	ClientGroups      string    `json:"client_groups"`
	StartDt           time.Time `json:"start_dt"`
	EventDt           time.Time `json:"event_dt"`
	EndDt             time.Time `json:"end_dt"`
	Ric               string    `json:"ric"`
	Relevance         string    `json:"relevance"`
	Country           string    `json:"country"`
	IndicatorName     string    `json:"indicator_name"`
	EventType         string    `json:"event_type"`
	ReutersPoll       string    `json:"reuters_poll"`
	PredictedSurprise string    `json:"predicted_surprise"`
	Actual            string    `json:"actual"`
	Unit              string    `json:"unit"`
	Surprise          string    `json:"surprise"`
	Prior             string    `json:"prior"`
}

type DataType []Data

type Response struct {
	Pagination Pagination `json:"pagination"`
	Data       DataType   `json:"data"`
}

type Option func(c *Client)

func SetAPIBaseURL(url string) Option {
	return func(news *Client) {
		news.APIBaseURL = url
	}
}

func SetHTTPClient(client *http.Client) Option {
	return func(news *Client) {
		news.Client = client
	}
}

func New(opts ...Option) *Client {
	service := &Client{}
	service.APIBaseURL = APIBaseURL
	service.Client = http.DefaultClient

	for _, opt := range opts {
		opt(service)
	}

	return service
}

func (news *Client) GetNews(dateFrom int64, dateTo int64) (Response, error) {
	newsPath := fmt.Sprintf("/api/v1/news/?offset=0&limit=20&from_dt=%d&till_dt=%d", dateFrom, dateTo)

	// create http request
	request, err := http.NewRequest(http.MethodGet, news.APIBaseURL+newsPath, nil)
	if err != nil {
		log.Error(err)
		return Response{}, err
	}

	// set content-type
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	// do request
	response, err := news.Client.Do(request)
	if err != nil {
		return Response{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error(err)
		}
	}(response.Body)

	// check response status
	if response.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("response status code: %d ", response.StatusCode)
	}

	// read response
	dataBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return Response{}, err
	}

	// parse response
	var newsResponse Response
	err = json.Unmarshal(dataBytes, &newsResponse)
	if err != nil {
		return Response{}, err
	}

	return newsResponse, nil
}

func isWeekEnd(t time.Time) bool {
	wd := t.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

func (nd *DataType) AddNasdaqOpening(holidays NasdaqHolidays) {
	t := time.Now().UTC()

	// market closed on weekend
	if isWeekEnd(t) {
		return
	}

	dt := utils.NasdaqOpeningTimeOfDay(t)
	title := "Nasdaq Opening"

	// check calendar for holiday
	if holiday := holidays.CheckDate(t); holiday != "" {
		title = "Nasdaq closed: " + holiday
	}
	*nd = append(*nd, Data{
		Title:      title,
		Symbol:     "USD",
		MmsSymbols: "*",
		DmSymbols:  "*",
		StartDt:    dt,
		EventDt:    dt,
		EndDt:      dt,
		Country:    "US",
		EventType:  "Trading hours",
	})
}

type SlackNews []SlackNewsItem

type SlackNewsItemKey struct {
	StartDt     time.Time
	StartDtUTC  string
	StartDtEET  string
	EventType   string
	Symbol      string
	DmSymbols   string
	IsImportant bool
}

type SlackNewsItem struct {
	SlackNewsItemKey
	Title string
}

func (nd *DataType) MergeForSlack(filter string) *SlackNews {

	// merge similar events
	m := make(map[SlackNewsItemKey][]string)
	for _, i := range *nd {

		// filter news by symbol
		ok, err := regexp.MatchString(filter, i.Symbol+" "+i.DmSymbols)
		if err != nil {
			log.Error(err)
		}
		if !ok {
			continue
		}

		// Present time for UTC and EET timezones
		eetLocation, err := time.LoadLocation("EET")
		if err != nil {
			log.Error(err)
		}
		startDtUTC := i.StartDt.Format("15:04")
		startDtEET := i.StartDt.In(eetLocation).Format("15:04")

		// Check if news item is important
		isImportant := false
		if i.Symbol == "USD" && i.DmSymbols == "*" {
			isImportant = true
		}
		key := SlackNewsItemKey{
			StartDt:     i.StartDt,
			StartDtUTC:  startDtUTC,
			StartDtEET:  startDtEET,
			EventType:   i.EventType,
			Symbol:      i.Symbol,
			DmSymbols:   i.DmSymbols,
			IsImportant: isImportant,
		}
		m[key] = append(m[key], i.Title)
	}
	res := make(SlackNews, 0, len(m))
	for key, titles := range m {
		res = append(res, SlackNewsItem{
			SlackNewsItemKey: key,
			Title:            strings.Join(titles, " | "),
		})
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].StartDt.Before(res[j].StartDt)
	})
	return &res
}

func (sn *SlackNews) RenderForSlack() (string, error) {
	if len(*sn) < 1 {
		return "", errors.New("no news to render")
	}
	t := template.New("slack")
	parse, err := t.Parse(slackTemplate)
	if err != nil {
		return "", err
	}
	var tpl bytes.Buffer
	if err := parse.Execute(&tpl, *sn); err != nil {
		return "", err
	}

	return tpl.String(), nil
}
