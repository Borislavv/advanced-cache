package helper

import (
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	localesandlanguages "gitlab.xbet.lan/v3group/backend/packages/go/locales-and-languages"
	"math/rand"
	"net/http"
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
			_ = lng
			for projectID := 1; projectID < 1000; projectID++ {
				if i > num {
					return list
				}
				//list = append(list, model.NewRequest(
				//	strconv.Itoa(projectID),
				//	"1xbet.com",
				//	string(lng),
				//	`{"name": "betting_`+strconv.Itoa(projectID*i)+`","choice": {"name": "betting_`+GenerateRandomString()+`_sport", "choice": null}}`,
				//))
				list = append(list, model.NewRequest(
					"285",
					"1xbet.com",
					"en",
					`{"name": "jared", "choice": "null"}`,
				))
				i++
			}
		}

		i++
	}
}

func GenerateRandomResponses(cfg *config.Response, num int) ([]*model.Response, error) {
	list := make([]*model.Response, 0, num)
	for _, req := range GenerateRandomRequests(num) {
		body := []byte(GenerateRandomString())
		resp, err := model.NewResponse(cfg, http.Header{}, req, body, func() ([]byte, error) {
			return body, nil
		})
		if err != nil {
			return nil, err
		}
		list = append(list, resp)
	}
	return list, nil
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
