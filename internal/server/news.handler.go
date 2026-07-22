package server

import (
	"log/slog"
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
		slog.Warn("[News] no news in DB for today, triggering Tavily", "error", err)

		tavilyNews, tErr := s.services.Tavily.SearchWeb(r.Context(), "Tin tức thị trường chứng khoán Việt Nam hôm nay", common.NEWS_DOMAINS)

		if tErr != nil {
			slog.Error("[News] failed to get latest news from Tavily", "error", tErr)
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
				slog.Error("[News] failed to save news to DB", "error", sErr)
				continue
			}
			latestNews = append(latestNews, saved)
		}
	}

	common.WriteJSON(w, http.StatusOK, latestNews)
}
