// Package handler provides HTTP endpoints for the article module.
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/kun/zhisuo-server/internal/article/usecase"
	"github.com/kun/zhisuo-server/internal/port"
)

// ArticleHandler wires Gin HTTP handlers to the article use case.
type ArticleHandler struct {
	uc *usecase.ArticleUseCase
}

// NewArticleHandler creates an ArticleHandler backed by the given use case.
func NewArticleHandler(uc *usecase.ArticleUseCase) *ArticleHandler {
	return &ArticleHandler{uc: uc}
}

// ListArticleRequest is the JSON body for listing articles (currently empty).
type ListArticleRequest struct{}

// List godoc
// @Summary      List all articles
// @Description  Returns every article in the system.
// @Tags         articles
// @Accept       json
// @Produce      json
// @Param        request body ListArticleRequest true "Request body (may be empty)"
// @Success      200  {object}  port.Response{data=[]entity.Article}
// @Router       /articles/list [post]
func (h *ArticleHandler) List(c *gin.Context) {
	articles, err := h.uc.List(c.Request.Context())
	if err != nil {
		port.ErrorInternal(c, err.Error())
		return
	}

	port.Success(c, articles)
}

// GetArticleRequest is the JSON body for fetching an article.
type GetArticleRequest struct {
	ID int64 `json:"id" binding:"required" example:"1" minimum:"1"`
}

// GetByID godoc
// @Summary      Get article by ID
// @Description  Returns a single article by its ID.
// @Tags         articles
// @Accept       json
// @Produce      json
// @Param        request body GetArticleRequest true "Article ID"
// @Success      200  {object}  port.Response{data=entity.Article}
// @Router       /articles/get [post]
func (h *ArticleHandler) GetByID(c *gin.Context) {
	var req GetArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		port.Error(c, port.CodeBadRequest, err.Error())
		return
	}

	article, err := h.uc.GetByID(c.Request.Context(), req.ID)
	if err != nil {
		if errors.Is(err, usecase.ErrArticleNotFound) {
			port.Error(c, port.CodeArticleNotFound, "article not found")
			return
		}
		port.ErrorInternal(c, err.Error())
		return
	}

	port.Success(c, article)
}

// CreateArticleRequest is the JSON body for creating an article.
type CreateArticleRequest struct {
	UserID  int64  `json:"user_id" binding:"required" example:"1" minimum:"1"`
	Title   string `json:"title" binding:"required" example:"My First Article" minLength:"1"`
	Content string `json:"content" binding:"required" example:"This is the article content." minLength:"1"`
}

// Create godoc
// @Summary      Create an article
// @Description  Creates a new article owned by the specified user.
// @Tags         articles
// @Accept       json
// @Produce      json
// @Param        request body CreateArticleRequest true "Article details"
// @Success      200  {object}  port.Response{data=entity.Article}
// @Router       /articles/create [post]
func (h *ArticleHandler) Create(c *gin.Context) {
	var req CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		port.Error(c, port.CodeBadRequest, err.Error())
		return
	}

	article, err := h.uc.Create(c.Request.Context(), req.UserID, req.Title, req.Content)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			port.Error(c, port.CodeUserNotFound, err.Error())
			return
		}
		port.ErrorInternal(c, err.Error())
		return
	}

	port.Success(c, article)
}

// UpdateArticleRequest is the JSON body for updating an article.
type UpdateArticleRequest struct {
	ID      int64  `json:"id" binding:"required" example:"1" minimum:"1"`
	Title   string `json:"title" binding:"required" example:"Updated Title" minLength:"1"`
	Content string `json:"content" binding:"required" example:"Updated content." minLength:"1"`
}

// Update godoc
// @Summary      Update an article
// @Description  Replaces the title and content of an existing article.
// @Tags         articles
// @Accept       json
// @Produce      json
// @Param        request body UpdateArticleRequest true "Article fields to update"
// @Success      200  {object}  port.Response{data=entity.Article}
// @Router       /articles/update [post]
func (h *ArticleHandler) Update(c *gin.Context) {
	var req UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		port.Error(c, port.CodeBadRequest, err.Error())
		return
	}

	article, err := h.uc.Update(c.Request.Context(), req.ID, req.Title, req.Content)
	if err != nil {
		if errors.Is(err, usecase.ErrArticleNotFound) {
			port.Error(c, port.CodeArticleNotFound, "article not found")
			return
		}
		port.ErrorInternal(c, err.Error())
		return
	}

	port.Success(c, article)
}

// DeleteArticleRequest is the JSON body for deleting an article.
type DeleteArticleRequest struct {
	ID int64 `json:"id" binding:"required" example:"1" minimum:"1"`
}

// Delete godoc
// @Summary      Delete an article
// @Description  Removes an article by its ID.
// @Tags         articles
// @Accept       json
// @Produce      json
// @Param        request body DeleteArticleRequest true "Article ID to delete"
// @Success      200  {object}  port.Response
// @Router       /articles/delete [post]
func (h *ArticleHandler) Delete(c *gin.Context) {
	var req DeleteArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		port.Error(c, port.CodeBadRequest, err.Error())
		return
	}

	if err := h.uc.Delete(c.Request.Context(), req.ID); err != nil {
		if errors.Is(err, usecase.ErrArticleNotFound) {
			port.Error(c, port.CodeArticleNotFound, "article not found")
			return
		}
		port.ErrorInternal(c, err.Error())
		return
	}

	port.Success(c, nil)
}
