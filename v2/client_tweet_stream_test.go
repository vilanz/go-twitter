package twitter

import (
	"context"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestClient_TweetSampleStream(t *testing.T) {
	type fields struct {
		Authorizer Authorizer
		Client     *http.Client
		Host       string
	}
	type args struct {
		opts TweetSampleStreamOpts
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       *TweetStream
		wantTweet  []*TweetMessage
		wantSystem []map[SystemMessageType]SystemMessage
		wantErr    bool
	}{
		{
			name: "Success",
			fields: fields{
				Authorizer: &mockAuth{},
				Host:       "https://www.test.com",
				Client: mockHTTPClient(func(req *http.Request) *http.Response {
					if req.Method != http.MethodGet {
						log.Panicf("the method is not correct %s %s", req.Method, http.MethodGet)
					}
					if strings.Contains(req.URL.String(), string(tweetSampleStreamEndpoint)) == false {
						log.Panicf("the url is not correct %s %s", req.URL.String(), tweetSampleStreamEndpoint)
					}
					stream := `{"data":{"id":"1","text":"hello"}, "matching_rules": [{ "id": "rule 1", "tag": "rule tag 1" }]}`
					stream += "\r\n"
					stream += `{"error":{"message":"Forced Disconnect: Too many connections. (Allowed Connections = 2)","sent":"2017-01-11T18:12:52+00:00"}}`
					stream += "\r\n"
					stream += `{"data":{"id":"2","text":"world"}, "matching_rules": [{ "id": "rule 2", "tag": "rule tag 2" }]}`
					stream += "\r\n"
					stream += "\r\n"
					stream += "\r\n"
					stream += "\r\n"
					stream += `{"data":{"id":"3","text":"!!"}}`
					stream += "\r\n"
					stream += `{"error":{"message":"Invalid date format for query parameter 'fromDate'. Expected format is 'yyyyMMddHHmm'. For example, '201701012315' for January 1st, 11:15 pm 2017 UTC.\n\n","sent":"2017-01-11T17:04:13+00:00"}}`
					stream += "\r\n"
					stream += `{"error":{"message":"Force closing connection to because it reached the maximum allowed backup (buffer size is ).","sent":"2017-01-11T17:04:13+00:00"}}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(stream)),
						Header: func() http.Header {
							h := http.Header{}
							h.Add(rateLimit, "15")
							h.Add(rateRemaining, "12")
							h.Add(rateReset, "1644461060")
							return h
						}(),
					}
				}),
			},
			args: args{
				opts: TweetSampleStreamOpts{},
			},
			wantTweet: []*TweetMessage{
				{
					Raw: &StreamedTweetRaw{
						Tweets: []*TweetObj{
							{
								ID:   "1",
								Text: "hello",
							},
						},
						MatchingRules: []*MatchingRule{
							{
								Id:  "rule 1",
								Tag: "rule tag 1",
							},
						},
					},
				},
				{
					Raw: &StreamedTweetRaw{
						Tweets: []*TweetObj{
							{
								ID:   "2",
								Text: "world",
							},
						},
						MatchingRules: []*MatchingRule{
							{
								Id:  "rule 2",
								Tag: "rule tag 2",
							},
						},
					},
				},
				{
					Raw: &StreamedTweetRaw{
						Tweets: []*TweetObj{
							{
								ID:   "3",
								Text: "!!",
							},
						},
					},
				},
			},
			wantSystem: []map[SystemMessageType]SystemMessage{
				{
					ErrorMessageType: {
						Message: "Forced Disconnect: Too many connections. (Allowed Connections = 2)",
						Sent: func() time.Time {
							t, _ := time.Parse(time.RFC3339, "2017-01-11T18:12:52+00:00")
							return t
						}(),
					},
				},
				{
					ErrorMessageType: {
						Message: "Invalid date format for query parameter 'fromDate'. Expected format is 'yyyyMMddHHmm'. For example, '201701012315' for January 1st, 11:15 pm 2017 UTC.\n\n",
						Sent: func() time.Time {
							t, _ := time.Parse(time.RFC3339, "2017-01-11T17:04:13+00:00")
							return t
						}(),
					},
				},
				{
					ErrorMessageType: {
						Message: "Force closing connection to because it reached the maximum allowed backup (buffer size is ).",
						Sent: func() time.Time {
							t, _ := time.Parse(time.RFC3339, "2017-01-11T17:04:13+00:00")
							return t
						}(),
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				Authorizer: tt.fields.Authorizer,
				Client:     tt.fields.Client,
				Host:       tt.fields.Host,
			}
			stream, err := c.TweetSampleStream(context.Background(), tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.TweetSampleStream() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			tweets := []*TweetMessage{}
			systems := []map[SystemMessageType]SystemMessage{}
			timer := time.NewTimer(time.Second * 5)

			func() {
				defer stream.Close()
				for {
					select {
					case sysMsg := <-stream.SystemMessages():
						systems = append(systems, sysMsg)
					case tweetMsg := <-stream.Tweets():
						tweets = append(tweets, tweetMsg)
					case <-timer.C:
						return
					case err := <-stream.Err():
						t.Errorf("Client.TweetSampleStream() error %v", err)
						return
					}
				}
			}()

			if !reflect.DeepEqual(tweets, tt.wantTweet) {
				t.Errorf("Client.TweetSampleStream() tweets = %v, want %v", tweets, tt.wantTweet)
			}
			if !reflect.DeepEqual(systems, tt.wantSystem) {
				t.Errorf("Client.TweetSampleStream() systems = %v, want %v", systems, tt.wantSystem)
			}
		})
	}
}
