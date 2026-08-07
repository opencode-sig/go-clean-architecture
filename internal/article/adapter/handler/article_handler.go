// Package handler provides HTTP endpoints for the article module.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kun/zhisuo-server/internal/article/usecase"
	"github.com/kun/zhisuo-server/internal/port"
)

// ArticleHandler wires Gin HTTP handlers to the article use case.
type ArticleHandler struct {
	uc      *usecase.ArticleUseCase
	pageCfg port.PageConfig
}

// NewArticleHandler creates an ArticleHandler backed by the given use case.
func NewArticleHandler(uc *usecase.ArticleUseCase, pageCfg port.PageConfig) *ArticleHandler {
	return &ArticleHandler{uc: uc, pageCfg: pageCfg}
}

// ListArticleRequest is the JSON body for listing articles.
type ListArticleRequest struct {
	Page     int `json:"page" example:"1" minimum:"1"`
	PageSize int `json:"page_size" example:"20" minimum:"1"`
}

// List godoc
// @Summary      List all articles
// @Description  Returns a page of articles, newest first.
// @Tags         articles
// @Accept       json
// @Produce      json
// @Param        request body ListArticleRequest true "Pagination (optional)"
// @Success      200  {object}  port.Response{data=port.Page}
// @Router       /articles/list [post]
func (h *ArticleHandler) List(c *gin.Context) {
	var req ListArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		port.Error(c, port.CodeBadRequest, err.Error())
		return
	}

	page, err := h.uc.List(c.Request.Context(), h.pageCfg.WithDefaults(req.Page, req.PageSize))
	if err != nil {
		port.ResponseError(c, err)
		return
	}

	port.Success(c, page)
}

// ListByUserRequest is the JSON body for listing articles by owner.
type ListByUserRequest struct {
	UserID   int64 `json:"user_id" binding:"required" example:"1" minimum:"1"`
	Page     int   `json:"page" example:"1" minimum:"1"`
	PageSize int   `json:"page_size" example:"20" minimum:"1"`
}

// ListByUser godoc
// @Summary      List articles by user
// @Description  Returns a page of articles owned by the given user.
// @Tags         articles
// @Accept       json
// @Produce      json
// @Param        request body ListByUserRequest true "User ID + pagination"
// @Success      200  {object}  port.Response{data=port.Page}
// @Router       /articles/by-user [post]
func (h *ArticleHandler) ListByUser(c *gin.Context) {
	var req ListByUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		port.Error(c, port.CodeBadRequest, err.Error())
		return
	}

	page, err := h.uc.ListByUser(c.Request.Context(), req.UserID, h.pageCfg.WithDefaults(req.Page, req.PageSize))
	if err != nil {
		port.ResponseError(c, err)
		return
	}

	port.Success(c, page)
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
		port.ResponseError(c, err)
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
		port.ResponseError(c, err)
		return
	}

	port.Success(c, article)
}

// UpdateArticleRequest is the JSON body for updating an article.
// Version is the optimistic-concurrency token from a prior read; omit (0) to skip the check.
type UpdateArticleRequest struct {
	ID      int64  `json:"id" binding:"required" example:"1" minimum:"1"`
	Version int64  `json:"version" example:"0" minimum:"0"`
	Title   string `json:"title" binding:"required" example:"Updated Title" minLength:"1"`
	Content string `json:"content" binding:"required" example:"Updated content." minLength:"1"`
}

// Update godoc
// @Summary      Update an article
// @Description  Replaces the title and content of an existing article (optimistic lock via version).
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

	article, err := h.uc.Update(c.Request.Context(), req.ID, req.Version, req.Title, req.Content)
	if err != nil {
		port.ResponseError(c, err)
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
		port.ResponseError(c, err)
		return
	}

	port.Success(c, nil)
}
