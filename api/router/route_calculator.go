package router

import (
	"log"
	"time"

	"github.com/4r7hur0/PBL-2/schemas" 
)

// IsValidCity verifica se a cidade está na lista de cidades conhecidas.

func IsValidCity(city string, citiesList []string) bool {
	for _, c := range citiesList {
		if c == city {
			return true
		}
	}
	return false
}

func findAllPathsDFS(origin, destination string, citiesList []string) [][]string {
	var paths [][]string
	var currentPath []string
	visited := make(map[string]bool)

	var dfs func(cityNode string)
	dfs = func(cityNode string) {
		currentPath = append(currentPath, cityNode)
		visited[cityNode] = true

		if cityNode == destination {
			pathCopy := make([]string, len(currentPath))
			copy(pathCopy, currentPath)
			paths = append(paths, pathCopy)
		} else {
			for _, neighbor := range citiesList {
				if !visited[neighbor] {
					dfs(neighbor)
				}
			}
		}

		if len(currentPath) > 0 {
			currentPath = currentPath[:len(currentPath)-1]
		}
		delete(visited, cityNode)
	}

	dfs(origin)
	return paths
}

func convertPathsToRouteSegments(paths [][]string) [][]schemas.RouteSegment {
	var routeSegmentsList [][]schemas.RouteSegment
	for _, path := range paths {
		var singleRoute []schemas.RouteSegment
		currentTime := time.Now().UTC() 
		for _, city := range path {
			segment := schemas.RouteSegment{
				City: city,
				ReservationWindow: schemas.ReservationWindow{
					StartTimeUTC: currentTime,
					EndTimeUTC:   currentTime.Add(1 * time.Minute),
				},
			}
			singleRoute = append(singleRoute, segment)
			currentTime = currentTime.Add(1 * time.Minute)
		}
		if len(singleRoute) > 0 {
			routeSegmentsList = append(routeSegmentsList, singleRoute)
		}
	}
	return routeSegmentsList
}

// GeneratePossibleRoutes é a função principal exportada para gerar as rotas.
// Ela recebe a lista de todas as cidades como parâmetro, tornando o pacote mais flexível.
func GeneratePossibleRoutes(origin, destination string, allCitiesList []string) [][]schemas.RouteSegment {
	if !IsValidCity(origin, allCitiesList) || !IsValidCity(destination, allCitiesList) {
		log.Printf("ROUTING: Origem '%s' ou Destino '%s' inválido(s) ou não consta(m) na lista de cidades.", origin, destination)
		return [][]schemas.RouteSegment{}
	}

	if origin == destination {
		utcNow := time.Now().UTC()
		segment := schemas.RouteSegment{
			City: origin,
			ReservationWindow: schemas.ReservationWindow{
				StartTimeUTC: utcNow,
				EndTimeUTC:   utcNow.Add(1 * time.Hour),
			},
		}
		return [][]schemas.RouteSegment{{segment}}
	}

	cityPaths := findAllPathsDFS(origin, destination, allCitiesList)
	if len(cityPaths) == 0 {
		log.Printf("ROUTING: Nenhum caminho encontrado entre '%s' e '%s' usando DFS.", origin, destination)
	}
	return convertPathsToRouteSegments(cityPaths)
}

