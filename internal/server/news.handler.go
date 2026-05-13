package server

import (
	"log"
	"net/http"
	"stockmind/internal/common"
	"stockmind/internal/database"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Server) GetNewsHandler(w http.ResponseWriter, r *http.Request) {
	currentDate := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	latestNews, err := s.queries.GetLatestNews(r.Context(), currentDate)

	if err != nil || len(latestNews) == 0 {
		log.Printf("[News] Failed to get latest news from DB or no news found for today (err: %v). Triggering Tavily...", err)

		tavilyNews, tErr := s.service.Tavily.SearchWeb(r.Context(), "Tin tức thị trường chứng khoán Việt Nam hôm nay", common.NEWS_DOMAINS)

		if tErr != nil {
			log.Printf("[News] Failed to get latest news from Tavily: %v", tErr)
			common.WriteJSONError(w, http.StatusInternalServerError, "Failed to get latest news")
			return
		}

		for _, article := range tavilyNews {
			saved, sErr := s.queries.SaveNews(r.Context(), database.SaveNewsParams{
				Title:       article.Title,
				Url:         article.URL,
				Description: article.Description,
			})
			if sErr != nil {
				log.Printf("[News] Failed to save news to DB: %v", sErr)
				continue
			}
			latestNews = append(latestNews, saved)
		}
	}

	common.WriteJSON(w, http.StatusOK, latestNews)
}
