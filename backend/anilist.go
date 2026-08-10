// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type AniListMediaInfo struct {
	ArtworkURL  string   `json:"artwork_url"`
	BannerURL   string   `json:"banner_url"`
	Description string   `json:"description"`
	Score       float64  `json:"score"`
	Genres      []string `json:"genres"`
	TrailerURL  string   `json:"trailer_url"`
}

var (
	htmlTagRegex  = regexp.MustCompile(`<[^>]*>`)
	cleanQueryReg = regexp.MustCompile(`(?i)\b(dublado|legendado|multi-audio|hd|fullhd|fhd|4k|1080p|720p|pt-br|ptbr|season\s*\d+|s\d+|ep\d+|episódio\s*\d+|ep\s*\d+)\b`)
)

func cleanAnimeSearchTitle(title string) string {
	cleaned := cleanQueryReg.ReplaceAllString(title, " ")
	cleaned = strings.ReplaceAll(cleaned, "(", " ")
	cleaned = strings.ReplaceAll(cleaned, ")", " ")
	cleaned = strings.ReplaceAll(cleaned, "[", " ")
	cleaned = strings.ReplaceAll(cleaned, "]", " ")
	cleaned = strings.ReplaceAll(cleaned, "-", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return cleaned
}

func FetchAniListMetadata(ctx context.Context, title string) (*AniListMediaInfo, error) {
	queryTitle := cleanAnimeSearchTitle(title)
	if queryTitle == "" {
		queryTitle = title
	}

	graphqlQuery := `
	query ($search: String) {
	  Media(search: $search, type: ANIME) {
	    coverImage {
	      extraLarge
	      large
	      medium
	    }
	    bannerImage
	    description(asHtml: false)
	    averageScore
	    genres
	    trailer {
	      id
	      site
	    }
	  }
	}`

	reqBody, err := json.Marshal(map[string]any{
		"query": graphqlQuery,
		"variables": map[string]any{
			"search": queryTitle,
		},
	})
	if err != nil {
		return nil, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, "https://graphql.anilist.co", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AniList returned HTTP %d", resp.StatusCode)
	}

	var response struct {
		Data struct {
			Media struct {
				CoverImage struct {
					ExtraLarge string `json:"extraLarge"`
					Large      string `json:"large"`
					Medium     string `json:"medium"`
				} `json:"coverImage"`
				BannerImage string   `json:"bannerImage"`
				Description string   `json:"description"`
				Score       float64  `json:"averageScore"`
				Genres      []string `json:"genres"`
				Trailer     struct {
					ID   string `json:"id"`
					Site string `json:"site"`
				} `json:"trailer"`
			} `json:"Media"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	media := response.Data.Media
	cover := media.CoverImage.ExtraLarge
	if cover == "" {
		cover = media.CoverImage.Large
	}
	if cover == "" {
		cover = media.CoverImage.Medium
	}

	desc := htmlTagRegex.ReplaceAllString(media.Description, "")
	desc = strings.TrimSpace(desc)

	trailerURL := ""
	if strings.EqualFold(media.Trailer.Site, "youtube") && media.Trailer.ID != "" {
		trailerURL = "https://www.youtube.com/watch?v=" + media.Trailer.ID
	}

	return &AniListMediaInfo{
		ArtworkURL:  cover,
		BannerURL:   media.BannerImage,
		Description: desc,
		Score:       media.Score / 10.0,
		Genres:      media.Genres,
		TrailerURL:  trailerURL,
	}, nil
}
