package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

// Costanti di programma
const (
	// Il banner grafico da stampare all'inizio del programma
	ProgramBanner string = "HyperWeather"

	// Dimensione vettori e numero chiamate per giornata
	RequiredApiCalls int = 480

	// Tempo di attesa da una chiamata e l'altra
	TimeToSleep = 3 * time.Minute

	// Posizione iniziale in ogni vettore dati meteo
	FirstPos int = 0
)

// load da .env
var (
	// API key, per chiamata alla stazione meteo
	APIkey   string
	CityCode string
)

// tipo per unmarshal dati meteo da stazione
type DatiMeteoChiamata struct {
	Weather []struct {
		Description string // descrizione meteo
	}
	Main struct {
		Pressure int     // millibar
		Temp     float64 // kelvin
		Humidity int     // in percentuale
	}
}

// Liste per dati meteo
var (
	vPressione   [RequiredApiCalls]float64
	vTemperatura [RequiredApiCalls]float64
	vUmidita     [RequiredApiCalls]int
	vCondizione  [RequiredApiCalls]string
)

// Esiti previsioni
var (
	esitoA bool = false
	esitoB bool = false
	esitoC bool = false
	esitoD bool = false
)

// Funzione per inizializzare i vettori
func initVettori() {
	for i := FirstPos; i < RequiredApiCalls; i++ {
		vPressione[i] = 1012.0
		vTemperatura[i] = 18.0
		vUmidita[i] = 65
		vCondizione[i] = "clear sky"
	}
}

// Funzione che inizializza variabili d'ambiente
func initEnvVar() {
	/*
		err := godotenv.Load()
		if err != nil {
			log.Fatal("errore caricamento variabili d'ambiente.")
		}*/
	APIkey = os.Getenv("APIkey")
	CityCode = os.Getenv("CityCode")
}

// Funzione per il calcolo della deviazione standard, per la pressione.
func devStandard(estremo int) float64 {
	var deviazione float64 = 0.0

	somma := 0.0
	// Calcolo media
	for i := RequiredApiCalls - 1; i >= estremo; i-- {
		somma += vPressione[i]
	}
	media := somma / float64(RequiredApiCalls)

	for i := RequiredApiCalls - 1; i >= estremo; i-- {
		deviazione += math.Pow(vPressione[i]-media, 2.0)
	}

	deviazione = math.Sqrt(deviazione / float64(RequiredApiCalls-estremo-1))

	return deviazione
}

/*
	Funzioni di previsione
*/

// Previsione di tipo A
// Temporale o pioggia forte entro la prossima ora
func previsioneA() {
	// Differenza di prezzione di 3.5 millibar in 9 minuti
	// var estr 9 minuti
	const estr int = 3

	if vPressione[estr]-vPressione[FirstPos] > 3.5 {
		esitoA = true
	} else {
		esitoA = false
	}
}

// Previsione di tipo B
// Pioggia leggera o intensa il giorna seguente
func previsioneB() {
	// Differenza di pressione di 3.5 in circa 5 ore
	// var estr circa 5 ore
	const estr int = 95

	if vPressione[estr]-vPressione[FirstPos] > 3.5 &&
		devStandard(estr) <= -4.0 &&
		vUmidita[FirstPos]-vUmidita[estr] >= 20 &&
		vTemperatura[estr]-vTemperatura[FirstPos] >= 2.0 &&
		vCondizione[FirstPos] != vCondizione[estr] {

		esitoB = true
	} else {
		esitoB = false
	}
}

// Previsione di tipo C
// Lungo raggia (5 giorni) con persistenza di una settimana
func previsioneC() {
	// Differenza di pressione di 3.5 in circa 24 ore
	// var estr 24 ore
	const estr = RequiredApiCalls - 1

	if vPressione[estr]-vPressione[FirstPos] > 3.5 &&
		devStandard(estr) <= 2.5 {

		esitoC = true
	} else {
		esitoC = false
	}
}

// Previsione di tipo D
// Miglioramento condizioni o staticità
func previsioneD() {
	// var estr 24 ore
	const estr int = RequiredApiCalls - 1
	if !esitoA && !esitoB && !esitoC &&
		devStandard(estr) <= 1.0 {

		esitoD = true
	} else {
		esitoD = false
	}
}

// Previsione
// Funzione che lancia in sequenza le previsioni
func previsioniMeteo() {
	previsioneA()
	previsioneB()
	previsioneC()
	previsioneD()
}

// Aggiornamento dati
// la funzione prende i dati (chiamata stazione) e aggiorna le liste (inserisce nuavi dati raccolti)
func aggiornaDatiMeteo() {
	// aggiorna liste
	// libera prima posizione
	for i := RequiredApiCalls - 1; i > FirstPos; i-- {
		vPressione[i] = vPressione[i-1]
		vTemperatura[i] = vTemperatura[i-1]
		vUmidita[i] = vUmidita[i-1]
		vCondizione[i] = vCondizione[i-1]
	}

	// chiamata al servizio meteo, raccoglie nuovi dati
	resp, err := http.Get("https://api.openweathermap.org/data/2.5/weather?q=" +
		CityCode + "&appid=" + APIkey)

	if err != nil {
		log.Println("errore chiamata alla stazione: no wifi?")
	}
	defer resp.Body.Close()

	// lettura response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("errore nella lettura dati response.")
	}

	// conversione json
	var datiMeteo DatiMeteoChiamata
	err = json.Unmarshal(body, &datiMeteo)
	if err != nil {
		log.Println("errore conversione json.")
		// chiamata fallita
		// inserisco dati dell'ultima chiamata funzionante
		vPressione[FirstPos] = vPressione[FirstPos+1]
		vTemperatura[FirstPos] = vTemperatura[FirstPos+1]
		vUmidita[FirstPos] = vUmidita[FirstPos+1]
		vCondizione[FirstPos] = vCondizione[FirstPos+1]
	} else { // chiamata funzionante
		// inserisco nuovi dati
		vPressione[FirstPos] = float64(datiMeteo.Main.Pressure)
		vTemperatura[FirstPos] = datiMeteo.Main.Temp - 273.15
		vUmidita[FirstPos] = datiMeteo.Main.Humidity
		vCondizione[FirstPos] = datiMeteo.Weather[0].Description
	}
}

func aggiornaMessaggioMeteo(w http.ResponseWriter, r *http.Request) {
	// scarto altri percorsi dall /
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// stampa banner
	fmt.Fprintf(w, ProgramBanner+"\n\n")
	fmt.Fprintf(w, `Dati meteo ultima chiamata: 
  - Pressione: %.0f millibar
  - Temperatura: %.2f °C 
  - Umidità: %d %%
  - Condizioni: %s.`,
		vPressione[0], vTemperatura[0], vUmidita[0], vCondizione[0])

	if esitoA || esitoB || esitoC || esitoD {
		fmt.Fprintf(w, "\n\nPrevisioni:\n")
	}
	if esitoA {
		fmt.Fprintf(w, " - Temporale entro un'ora.\n")
	}
	if esitoB {
		fmt.Fprintf(w, " - Domani pioggia.\n")
	}
	if esitoC {
		fmt.Fprintf(w, " - Tra 5 giorno maltempo per qualche giorno.\n")
	}
	if esitoD {
		fmt.Fprintf(w, " - Tempo in miglioramento.\n")
	}
}

func main() {
	// init vettori dati meteo
	initVettori()

	// init varibili d'ambiente
	initEnvVar()

	// stampa messaggio
	log.Printf(ProgramBanner + " is starting...")

	// avvio go routine per previsioni meteo
	go func() {
		for {
			// raccolta dati, chiamata e inserimento dati in liste
			aggiornaDatiMeteo()

			// elabora previsione
			previsioniMeteo()

			// attesa di 3 minuti dalla prossima chiamata
			time.Sleep(TimeToSleep)
		}
	}()

	// avvio web server

	// init funzione web server
	http.HandleFunc("/", aggiornaMessaggioMeteo)
	// init porta
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	// lancio web server
	log.Printf("Listening on port %s", port)
	log.Printf("Open http://localhost:%s in the browser", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
