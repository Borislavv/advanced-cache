package mock

import (
	"context"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	localesandlanguages "gitlab.xbet.lan/v3group/backend/packages/go/locales-and-languages"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	minStrLen = 8
	maxStrLen = 1024
)

func GenerateRandomRequests(num int) []*model.Request {
	list := make([]*model.Request, 0, num)

	i := 0
	for {
		for _, lng := range localesandlanguages.WebnameList() {
			for projectID := 1; projectID < 1000; projectID++ {
				if i > num {
					return list
				}
				list = append(list, model.NewRequest(
					[]byte(strconv.Itoa(projectID)),
					[]byte("california-sunshine.com"),
					[]byte(lng),
					[][]byte{
						[]byte(`betting_` + strconv.Itoa(projectID)),
						[]byte(`betting_null`),
						[]byte(`betting_null_sport`),
						[]byte(`betting_null_sport_` + strconv.Itoa(i)),
						[]byte(`betting_null_sport_` + strconv.Itoa(i) + `_` + strconv.Itoa(i*i)),
						[]byte(`betting_null_sport_` + strconv.Itoa(i) + `_` + strconv.Itoa(i*i) + `_` + strconv.Itoa(projectID*i)),
					},
				))
				i++
			}
		}

		i++
	}
}

func GenerateRandomResponses(cfg *config.Config, num int) []*model.Response {
	headers := http.Header{}
	headers.Add("Accept", "application/json")
	headers.Add("Content-Type", "application/json")

	list := make([]*model.Response, 0, num)
	for _, req := range GenerateRandomRequests(num) {
		data := model.NewData(200, headers, []byte(GenerateRandomString()))
		resp, err := model.NewResponse(data, req, cfg, func(ctx context.Context) (data *model.Data, err error) {
			return data, nil
		})
		if err != nil {
			panic(err)
		}
		list = append(list, resp)
	}
	return list
}

func GenerateRandomString() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	rand.Seed(time.Now().UnixNano())

	length := rand.Intn(maxStrLen-minStrLen+1) + minStrLen

	var sb strings.Builder
	sb.Grow(length)

	for i := 0; i < length; i++ {
		sb.WriteByte(letters[rand.Intn(len(letters))])
	}

	return sb.String()
}
