// Package handler provides HTTP handlers for comment endpoints.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kun/zhisuo-server/internal/comment/usecase"
	"github.com/kun/zhisuo-server/internal/port"
)

// CommentHandler exposes comment CRUD operations as HTTP endpoints.
type CommentHandler struct {
	uc      *usecase.CommentUseCase
	pageCfg port.PageConfig
}

// NewCommentHandler creates a CommentHandler backed by the given use case.
func NewCommentHandler(uc *usecase.CommentUseCase, pageCfg port.PageConfig) *CommentHandler {
	return &CommentHandler{uc: uc, pageCfg: pageCfg}
}

// ListCommentsRequest carries the article ID for listing comments.
type ListCommentsRequest struct {
	ArticleID int64 `json:"article_id" binding:"required" example:"1" minimum:"1"`
	Page      int   `json:"page" example:"1" minimum:"1"`
	PageSize  int   `json:"page_size" example:"20" minimum:"1"`
}

// ListByArticle godoc
// @Summary      List comments by article
// @Description  Returns a page of comments for a given article.
// @Tags         comments
// @Accept       json
// @Produce      json
// @Param        request body ListCommentsRequest true "Article ID + pagination"
// @Success      200  {object}  port.Response{data=port.Page}
// @Router       /comments/list [post]
func (h *CommentHandler) ListByArticle(c *gin.Context) {
	var req ListCommentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		port.Error(c, port.CodeBadRequest, err.Error())
		return
	}

	page, err := h.uc.ListByArticle(c.Request.Context(), req.ArticleID, h.pageCfg.WithDefaults(req.Page, req.PageSize))
	if err != nil {
		port.ResponseError(c, err)
		return
	}

	port.Success(c, page)
}

// CreateCommentRequest carries the JSON fields required to create a comment.
type CreateCommentRequest struct {
	ArticleID int64  `json:"article_id" binding:"required" example:"1" minimum:"1"`
	UserID    int64  `json:"user_id" binding:"required" example:"1" minimum:"1"`
	Content   string `json:"content" binding:"required" example:"Great article!" minLength:"1"`
}

// Create godoc
// @Summary      Create a comment
// @Description  Adds a new comment on the specified article.
// @Tags         comments
// @Accept       json
// @Produce      json
// @Param        request body CreateCommentRequest true "Comment details"
// @Success      200  {object}  port.Response{data=entity.Comment}
// @Router       /comments/create [post]
func (h *CommentHandler) Create(c *gin.Context) {
	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		port.Error(c, port.CodeBadRequest, err.Error())
		return
	}

	comment, err := h.uc.Create(c.Request.Context(), req.ArticleID, req.UserID, req.Content)
	if err != nil {
		port.ResponseError(c, err)
		return
	}

	port.Success(c, comment)
}

// DeleteCommentRequest carries the comment ID for deletion.
type DeleteCommentRequest struct {
	ID int64 `json:"id" binding:"required" example:"1" minimum:"1"`
}

// Delete godoc
// @Summary      Delete a comment
// @Description  Removes a comment by its ID.
// @Tags         comments
// @Accept       json
// @Produce      json
// @Param        request body DeleteCommentRequest true "Comment ID to delete"
// @Success      200  {object}  port.Response
// @Router       /comments/delete [post]
func (h *CommentHandler) Delete(c *gin.Context) {
	var req DeleteCommentRequest
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
