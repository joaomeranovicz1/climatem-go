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

type GeoResponse struct {
	Results []struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Admin1    string  `json:"admin1"`
		Country   string  `json:"country"`
	} `json:"results"`
}

type WeatherResponse struct {
	Current struct {
		Temperature2m       float64 `json:"temperature_2m"`
		RelativeHumidity2m  int     `json:"relative_humidity_2m"`
		ApparentTemperature float64 `json:"apparent_temperature"`
		WeatherCode         int     `json:"weather_code"`
		IsDay               int     `json:"is_day"`
		Precipitation       float64 `json:"precipitation"`
		WindSpeed           float64 `json:"wind_speed_10m"`
	} `json:"current"`
	Daily struct {
		UVIndexMax []float64 `json:"uv_index_max"`
	} `json:"daily"`
}

type AirQualityResponse struct {
	Current struct {
		BirchPollen   float64 `json:"birch_pollen"`
		GrassPollen   float64 `json:"grass_pollen"`
		OlivePollen   float64 `json:"olive_pollen"`
		RagweedPollen float64 `json:"ragweed_pollen"`
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
			http.Error(w, "Erro no HTML: "+err.Error(), 500)
			return
		}
		tmpl.Execute(w, nil)
	})

	http.HandleFunc("/api/clima", buscaDadosHandler)
	http.HandleFunc("/api/cidade", buscarCidadeHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		fmt.Println("🚀 Servidor Go rodando localmente em: http://localhost:8080")
	}

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func buscarCidadeHandler(w http.ResponseWriter, r *http.Request) {
	nome := r.URL.Query().Get("nome")
	if nome == "" {
		http.Error(w, "Nome vazio", 400)
		return
	}

	nomeSeguro := url.QueryEscape(nome)
	link := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=5&language=pt&format=json", nomeSeguro)

	var geo GeoResponse
	fetchJSON(link, &geo)

	w.Header().Set("Content-Type", "application/json")

	if len(geo.Results) > 0 {
		json.NewEncoder(w).Encode(geo.Results[0])
	} else {
		http.Error(w, "Cidade não encontrada", 404)
	}
}

func buscaDadosHandler(w http.ResponseWriter, r *http.Request) {
	lat := r.URL.Query().Get("lat")
	lon := r.URL.Query().Get("lon")

	urlClima := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%s&longitude=%s&current=temperature_2m,relative_humidity_2m,apparent_temperature,is_day,weather_code,precipitation,wind_speed_10m&daily=uv_index_max&timezone=auto", lat, lon)
	var dadosClima WeatherResponse
	fetchJSON(urlClima, &dadosClima)

	urlPolen := fmt.Sprintf("https://air-quality-api.open-meteo.com/v1/air-quality?latitude=%s&longitude=%s&current=birch_pollen,grass_pollen,olive_pollen,ragweed_pollen&timezone=auto", lat, lon)
	var dadosAr AirQualityResponse
	fetchJSON(urlPolen, &dadosAr)

	desc, icone := traduzirClima(dadosClima.Current.WeatherCode, dadosClima.Current.IsDay)
	_, textoPolen := calcularNivelPolen(dadosAr)

	uv := 0.0
	if len(dadosClima.Daily.UVIndexMax) > 0 {
		uv = dadosClima.Daily.UVIndexMax[0]
	}

	fraseDica, tipoDica := gerarDicaEsporte(dadosClima.Current.Temperature2m, dadosClima.Current.Precipitation, dadosClima.Current.WindSpeed, uv, 0)

	resultado := DadosFrontend{
		Temperatura: dadosClima.Current.Temperature2m,
		Sensacao:    dadosClima.Current.ApparentTemperature,
		Umidade:     dadosClima.Current.RelativeHumidity2m,
		UV:          uv,
		Polen:       textoPolen,
		Descricao:   desc,
		Icone:       icone,
		Dica:        fraseDica,
		TipoDica:    tipoDica,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resultado)
}

func fetchJSON(urlLink string, target interface{}) {
	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", urlLink, nil)
	if err != nil {
		fmt.Println("Erro req:", err)
		return
	}

	req.Header.Set("User-Agent", "ClimaTem-Estudante-BR/1.0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Erro de Rede:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		fmt.Println("API pediu calma (429). IP congestionado.")
		return
	}

	if resp.StatusCode != 200 {
		fmt.Println("API Rejeitou:", resp.Status)
		return
	}

	json.NewDecoder(resp.Body).Decode(target)
}

func gerarDicaEsporte(temp, chuva, vento, uv float64, nivelPolen int) (string, string) {
	if chuva >= 2.0 {
		return "🌧️ Chuva Forte: Evite treino outdoor.", "perigo"
	}
	if chuva > 0.1 {
		return "🌦️ Chuva Leve: Cuidado com piso liso.", "atencao"
	}
	if temp > 32 {
		return "🔥 Calor Extremo: Hidrate-se muito.", "perigo"
	}
	if uv >= 8 {
		return "☀️ UV Crítico: Use proteção total.", "perigo"
	}
	if vento > 35 {
		return "💨 Ventania: Risco de acidentes.", "perigo"
	}
	return "✅ Condições Perfeitas para treino!", "bom"
}

func calcularNivelPolen(ar AirQualityResponse) (int, string) {
	total := ar.Current.BirchPollen + ar.Current.GrassPollen + ar.Current.OlivePollen + ar.Current.RagweedPollen
	if total == 0 {
		return 0, "Indisponível 🚫"
	}
	if total > 50 {
		return 3, "Alto ⚠️"
	}
	if total > 20 {
		return 2, "Médio 🌾"
	}
	return 1, "Baixo ✅"
}

func traduzirClima(code, isDay int) (string, string) {
	switch code {
	case 0:
		if isDay == 1 {
			return "Céu Limpo", "☀️"
		} else {
			return "Céu Limpo", "🌙"
		}
	case 1, 2, 3:
		return "Nublado", "☁️"
	case 45, 48:
		return "Nevoeiro", "🌫️"
	case 51, 53, 55, 61, 63, 65:
		return "Chuva", "🌧️"
	case 80, 81, 82:
		return "Pancadas", "🌦️"
	case 95, 96, 99:
		return "Tempestade", "⚡"
	default:
		return "Nublado", "☁️"
	}
}
