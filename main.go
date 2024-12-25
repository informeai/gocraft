package main

import (
	"flag"
	"log"
	"time"

	_ "image/png"

	"net/http"
	_ "net/http/pprof"

	_ "embed"

	"github.com/faiface/mainthread"
	"github.com/informeai/gocraft/internal"
	"github.com/informeai/gocraft/scene"
	_ "github.com/informeai/gocraft/shader"
)

func run() {
	err := internal.LoadTextureDesc()
	if err != nil {
		log.Fatal(err)
	}

	err = scene.InitStore()
	if err != nil {
		log.Panic(err)
	}
	store := scene.GetStore()
	defer store.Close()

	err = scene.InitClient()
	if err != nil {
		log.Panic(err)
	}
	client := scene.GetClient()
	if client != nil {
		defer client.Close()
	}

	game, err := scene.NewGame(800, 600)
	if err != nil {
		log.Panic(err)
	}

	game.Camera.Restore(store.GetPlayerState())
	tick := time.Tick(time.Second / 60)
	for !game.ShouldClose() {
		<-tick
		game.Update()
	}
	store.UpdatePlayerState(game.Camera.State())
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	flag.Parse()
	go func() {
		if *scene.PprofPort != "" {
			log.Fatal(http.ListenAndServe(*scene.PprofPort, nil))
		}
	}()
	mainthread.Run(run)
}
