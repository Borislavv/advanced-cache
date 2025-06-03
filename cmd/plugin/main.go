package main

import (
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/model"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/storage"
)

func init() {
	//zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	//zerolog.SetGlobalLevel(zerolog.DebugLevel)
	//log.Logger = log.Output(os.Stdout)
	//
	//store = storage.New(config.Storage{
	//	EvictionAlgo:        string(algo.LRU),
	//	MemoryFillThreshold: 0.95,
	//	MemoryLimit:         1024 * 1024 * 10,
	//})
	//
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	//
	//seoRepo := repository.NewSeo()
	//
	//cfg := config.Response{
	//	RevalidateBeta:     0.5,
	//	RevalidateInterval: time.Minute * 10,
	//}
	//
	//reqs = make([]*model.Request, 0, 10000)
	//for i := 0; i < 10000; i++ {
	//	req := model.NewRequest("285", "1xbet.com", "en", `{"name": "betting", "choice": null}`+strconv.Itoa(i))
	//	resp, err := model.NewResponse(cfg, http.Header{}, req, []byte(`{"data": "success"}`), func() ([]byte, error) {
	//		return seoRepo.PageData()
	//	})
	//	if err != nil {
	//		panic(err)
	//	}
	//	store.Set(ctx, resp)
	//	reqs = append(reqs, req)
	//}
}

func main() {
	//go func() {
	//	if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
	//		log.Fatal().Err(err).Msg("Failed to start server")
	//	}
	//}()
	//
	//var i int
	//for {
	//	if i >= 10000 {
	//		i = 0
	//	}
	//	store.Get(reqs[i%10000])
	//	i++
	//}
}

func getData(ctx context.Context, db storage.Storage, seoRepo repository.Seo, req *model.Request) {
	_, _, _, _, err := db.Get(ctx, req, seoRepo.PageData)
	if err != nil {
		log.Err(err).Msg("failed to get body")
		return
	}

	_, _, _, _, err = db.Get(ctx, req, seoRepo.PageData)
	if err != nil {
		log.Err(err).Msg("failed to get body")
		return
	}

	_, _, _, _, err = db.Get(ctx, req, seoRepo.PageData)
	if err != nil {
		log.Err(err).Msg("failed to get body")
		return
	}

	_, _, _, _, err = db.Get(ctx, req, seoRepo.PageData)
	if err != nil {
		log.Err(err).Msg("failed to get body")
		return
	}

	_, _, _, _, err = db.Get(ctx, req, seoRepo.PageData)
	if err != nil {
		log.Err(err).Msg("failed to get body")
		return
	}
	//fmt.Println(string(data))
}
