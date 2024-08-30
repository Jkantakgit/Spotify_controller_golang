/**
*Project name: Spotify controller
*Package main
*File: main.go
*Date: 30.8.2024
*Last change: 30.8.2024
*Author: Petr Holánek
*
*/
package main


// Imports
import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/zmb3/spotify"
)

// initialize variables
var (
	ch        = make(chan *spotify.Client)
	auth      = spotify.NewAuthenticator("http://localhost:8080/callback", spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)
	state     = "state"
	broker    = "10.180.0.9:1883"
	topic     = []string{"zigbee2mqtt/Dvere Petr", "zigbee2mqtt/tlacitko"}
	client_id = "door_app"
)

// main function
func main() {
	client := initspotify()                 //init spotify client
	myWindow, content := initwindow(client) //init window and content
	go change_picture(client, content)      //start function to change picture
	myWindow.ShowAndRun()                   //run GUI
}

// init window and content
func initwindow(client *spotify.Client) (fyne.Window, *fyne.Container) {
	myApp := app.New() //create new app
	myWindow := myApp.NewWindow("Spotify Player")
	prev := widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() { //make skip to previous button
		client.Previous()
	})
	stop_play := widget.NewButtonWithIcon("", theme.MediaPauseIcon(), func() { //make stop/play button
		playback, err := client.PlayerState() // check if player is playing
		if err != nil {
			log.Fatal(err)
		}
		state := playback.Playing
		if state {
			go client.Pause() //pause if playing
		} else {
			go client.Play() //play if not playing
		}
	})
	next := widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() { //make skip to next button
		client.Next()
	})
	var ID *spotify.ID                                                         //init spotify ID
	pic, _ := song_picture(client, ID)                                         //init picture
	progress := initprogressbar(client, stop_play)                             //init progress bar
	controlBox := container.New(layout.NewHBoxLayout(), prev, stop_play, next) //make control box
	content := container.NewWithoutLayout(pic, progress, controlBox)           //put content together
	pic.Resize(fyne.NewSize(64, 64))
	pic.Move(fyne.NewPos(-5, -35))
	progress.Resize(fyne.NewSize(530, 5))
	progress.Move(fyne.NewPos(64, 0))
	controlBox.Resize(fyne.NewSize(20, 20))
	controlBox.Move(fyne.NewPos(250, 5))
	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(600, 60))
	myWindow.SetFixedSize(true)
	return myWindow, content
}

// initialize spotify client
func initspotify() *spotify.Client {
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	})
	go http.ListenAndServe(":8080", nil)

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)
	client := <-ch
	return client
}

// complete authorization for Spotify API requests
func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	client := auth.NewClient(tok)
	fmt.Fprintf(w, "Login Completed!")
	ch <- &client
}

// initialize progress bar for song progression
func initprogressbar(client *spotify.Client, button *widget.Button) *widget.ProgressBar {
	pbar := widget.NewProgressBar()
	go func() {
		for {
			playback, err := client.PlayerState()
			if err != nil {
				log.Fatal(err)
			}
			state := playback.Playing
			if state {
				button.SetIcon(theme.MediaPauseIcon())
			} else {
				button.SetIcon(theme.MediaPlayIcon())
			}
			if playback.Item == nil || !playback.Playing {
				time.Sleep(time.Millisecond * 100)
			} else {
				time.Sleep(time.Millisecond * 1)
				max := playback.Item.Duration
				pbar.SetValue(float64(playback.Progress) / float64(max))
				pbar.Refresh()
			}
		}
	}()
	return pbar
}

// initialize picture of current playing song
// if no song is playing use blank picture instead
func song_picture(client *spotify.Client, ID *spotify.ID) (*canvas.Image, *spotify.ID) {
	playback, err := client.PlayerState()
	if err != nil {
		log.Fatal(err)
		return canvas.NewImageFromFile("./resources/black.png"), ID
	}
	if !playback.Playing {
		time.Sleep(time.Second * 1)
	} else {
		time.Sleep(time.Millisecond * 100)
	}
	if playback.Item != nil && len(playback.Item.Album.Images) > 0 && playback.Item.ID != *ID {
		imageURL := playback.Item.Album.Images[2].URL
		resp, err := http.Get(imageURL)
		if err != nil {
			log.Fatal(err)
			return canvas.NewImageFromFile("./resources/black.png"), ID
		}
		defer resp.Body.Close()

		imageData, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
			return canvas.NewImageFromFile("./resources/black.png"), ID
		}

		imgResource := fyne.NewStaticResource("album_image.jpg", imageData)
		image := canvas.NewImageFromResource(imgResource)
		image.FillMode = canvas.ImageFillOriginal
		ID := playback.Item.ID
		return image, &ID
	}
	return canvas.NewImageFromFile("./resources/black.png"), ID
}

// helper function for changing the image of song
func change_picture(client *spotify.Client, content *fyne.Container) {
	var ID *spotify.ID
	var image *canvas.Image
	for {
		image, ID = song_picture(client, ID)
		image.Resize(fyne.NewSize(64, 64))
		content.Objects[0] = image
	}
}

// initialize MQTT client for subscribing to MQTT topics
func init_mqtt_client(spot *spotify.Client) {
	opts := mqtt.NewClientOptions().AddBroker(broker).SetClientID(client_id)
	opts.SetAutoReconnect(true)
	opts.DefaultPublishHandler = func(client mqtt.Client, msg mqtt.Message) {
		for _, topic := range topic {
			if string(msg.Topic()) == topic {
				fmt.Printf("Message arrived: %s\n", string(msg.Payload()))
			}
		}
	}
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
		os.Exit(1)
	}
	for _, topic := range topic {
		if token := client.Subscribe(topic, 1, nil); token.Wait() && token.Error() != nil {
			fmt.Printf("Error subscribing to topic %s: %s\n", topic, token.Error())
			continue
		}
		fmt.Printf("Subscribed to topic: %s\n", topic)
	}
}
