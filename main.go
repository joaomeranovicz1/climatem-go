package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const API_KEY_DEFAULT = "d497fb5bf4aa4ca99cb142839261003"

type SearchResponse []struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Region  string  `json:"region"`
	Country string  `json:"country"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Url     string  `json:"url"`
}

type WeatherAPIResponse struct {
	Location struct {
		Name   string `json:"name"`
		Region string `json:"region"`
	} `json:"location"`
	Current struct {
		TempC     float64 `json:"temp_c"`
		IsDay     int     `json:"is_day"`
		Condition struct {
			Text string `json:"text"`
			Icon string `json:"icon"`
			Code int    `json:"code"`
		} `json:"condition"`
		WindKph    float64 `json:"wind_kph"`
		Humidity   int     `json:"humidity"`
		FeelslikeC float64 `json:"feelslike_c"`
		UV         float64 `json:"uv"`
		PrecipMm   float64 `json:"precip_mm"`
	} `json:"current"`
}

type DadosFrontend struct {
	Temperatura float64 `json:"temp"`
	Sensacao    float64 `json:"sensacao"`
	Umidade     int     `json:"umidade"`
	UV          float64 `json:"uv"`
	Polen       string  `json:"polen"`
	Descricao   string  `json:"descricao"`
	Icone       string  `json:"icone"`
	Dica        string  `json:"dica"`
	TipoDica    string  `json:"tipo_dica"`
}

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		tmpl.Execute(w, nil)
	})

	http.HandleFunc("/api/clima", buscaDadosHandler)
	http.HandleFunc("/api/cidade", buscarCidadeHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getAPIKey() string {
	key := os.Getenv("WEATHER_API_KEY")
	if key == "" {
		return API_KEY_DEFAULT
	}
	return key
}

func buscarCidadeHandler(w http.ResponseWriter, r *http.Request) {
	nome := r.URL.Query().Get("nome")
	if nome == "" {
		http.Error(w, "Nome vazio", 400)
		return
	}

	apiKey := getAPIKey()
	nomeSeguro := url.QueryEscape(nome)
	link := fmt.Sprintf("http://api.weatherapi.com/v1/search.json?key=%s&q=%s", apiKey, nomeSeguro)

	var resultados SearchResponse
	if err := fetchJSON(link, &resultados); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	type GeoFrontend struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Admin1    string  `json:"admin1"`
		Country   string  `json:"country"`
	}

	w.Header().Set("Content-Type", "application/json")

	if len(resultados) > 0 {
		item := resultados[0]
		resp := GeoFrontend{
			Name:      item.Name,
			Latitude:  item.Lat,
			Longitude: item.Lon,
			Admin1:    item.Region,
			Country:   item.Country,
		}
		json.NewEncoder(w).Encode(resp)
	} else {
		http.Error(w, "Cidade não encontrada", 404)
	}
}

func buscaDadosHandler(w http.ResponseWriter, r *http.Request) {
	lat := r.URL.Query().Get("lat")
	lon := r.URL.Query().Get("lon")
	apiKey := getAPIKey()

	link := fmt.Sprintf("http://api.weatherapi.com/v1/current.json?key=%s&q=%s,%s&lang=pt&aqi=no", apiKey, lat, lon)

	var dados WeatherAPIResponse
	if err := fetchJSON(link, &dados); err != nil {
		http.Error(w, "Erro ao buscar clima", 500)
		return
	}

	desc, icone := traduzirCondicao(dados.Current.Condition.Code, dados.Current.IsDay)
	fraseDica, tipoDica := gerarDicaEsporte(
		dados.Current.TempC,
		dados.Current.PrecipMm,
		dados.Current.WindKph,
		dados.Current.UV,
	)

	resultado := DadosFrontend{
		Temperatura: dados.Current.TempC,
		Sensacao:    dados.Current.FeelslikeC,
		Umidade:     dados.Current.Humidity,
		UV:          dados.Current.UV,
		Polen:       "Baixo ✅",
		Descricao:   desc,
		Icone:       icone,
		Dica:        fraseDica,
		TipoDica:    tipoDica,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resultado)
}

func fetchJSON(urlLink string, target interface{}) error {
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(urlLink)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status api: %s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func gerarDicaEsporte(temp, chuva, vento, uv float64) (string, string) {
	if chuva >= 2.0 {
		return "🌧️ Chuva: Evite treino outdoor.", "perigo"
	}
	if temp > 32 {
		return "🔥 Calor Extremo: Hidrate-se!", "perigo"
	}
	if uv >= 8 {
		return "☀️ UV Crítico: Use proteção total.", "perigo"
	}
	if vento > 35 {
		return "💨 Ventania: Risco de acidentes.", "perigo"
	}
	if temp < 5 {
		return "❄️ Muito Frio: Agasalhe-se bem.", "atencao"
	}
	return "✅ Condições Perfeitas para treino!", "bom"
}

func traduzirCondicao(code int, isDay int) (string, string) {
	switch code {
	case 1000:
		if isDay == 1 {
			return "Céu Limpo", "☀️"
		}
		return "Céu Limpo", "🌙"
	case 1003:
		return "Parc. Nublado", "⛅"
	case 1006, 1009:
		return "Nublado", "☁️"
	case 1030, 1135, 1147:
		return "Neblina", "🌫️"
	case 1063, 1180, 1183, 1150, 1153:
		return "Chuva Leve", "🌦️"
	case 1186, 1189, 1192, 1195, 1240, 1243:
		return "Chuva", "🌧️"
	case 1273, 1276, 1087:
		return "Tempestade", "⛈️"
	default:
		return "Nublado", "☁️"
	}
}
