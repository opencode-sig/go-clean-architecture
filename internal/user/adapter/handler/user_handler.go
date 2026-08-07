// Package handler provides HTTP handlers for the user module.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kun/zhisuo-server/internal/port"
	"github.com/kun/zhisuo-server/internal/user/usecase"
)

// UserHandler exposes user operations as HTTP endpoints via Gin.
type UserHandler struct {
	uc      *usecase.UserUseCase
	pageCfg port.PageConfig
}

// NewUserHandler creates a UserHandler backed by the given use case.
func NewUserHandler(uc *usecase.UserUseCase, pageCfg port.PageConfig) *UserHandler {
	return &UserHandler{uc: uc, pageCfg: pageCfg}
}

// ListUserRequest is the JSON body for listing users.
type ListUserRequest struct {
	Page     int `json:"page" example:"1" minimum:"1"`
	PageSize int `json:"page_size" example:"20" minimum:"1"`
}

// List godoc
// @Summary      List all users
// @Description  Returns a page of registered users, newest first.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body ListUserRequest true "Pagination (optional)"
// @Success      200  {object}  port.Response{data=port.Page}
// @Router       /users/list [post]
func (h *UserHandler) List(c *gin.Context) {
	var req ListUserRequest
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

// GetUserRequest is the JSON body for fetching a user.
type GetUserRequest struct {
	ID int64 `json:"id" binding:"required" example:"1" minimum:"1"`
}

// GetByID godoc
// @Summary      Get user by ID
// @Description  Returns a single user by their unique ID.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body GetUserRequest true "User ID"
// @Success      200  {object}  port.Response{data=entity.User}
// @Router       /users/get [post]
func (h *UserHandler) GetByID(c *gin.Context) {
	var req GetUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		port.Error(c, port.CodeBadRequest, err.Error())
		return
	}

	user, err := h.uc.GetByID(c.Request.Context(), req.ID)
	if err != nil {
		port.ResponseError(c, err)
		return
	}

	port.Success(c, user)
}

// CreateUserRequest is the JSON body for creating a new user.
type CreateUserRequest struct {
	Username string `json:"username" binding:"required" example:"jane_doe" minLength:"1"`
	Email    string `json:"email" binding:"required" example:"jane@example.com"`
	Bio      string `json:"bio" example:"Software engineer and writer."`
}

// Create godoc
// @Summary      Create a user
// @Description  Registers a new user with the given username, email, and optional bio.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body CreateUserRequest true "User details"
// @Success      200  {object}  port.Response{data=entity.User}
// @Router       /users/create [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		port.Error(c, port.CodeBadRequest, err.Error())
		return
	}

	user, err := h.uc.Create(c.Request.Context(), req.Username, req.Email, req.Bio)
	if err != nil {
		port.ResponseError(c, err)
		return
	}

	port.Success(c, user)
}

// UpdateUserRequest is the JSON body for updating an existing user.
// Version is the optimistic-concurrency token from a prior read; omit (0) to skip the check.
type UpdateUserRequest struct {
	ID       int64  `json:"id" binding:"required" example:"1" minimum:"1"`
	Version  int64  `json:"version" example:"0" minimum:"0"`
	Username string `json:"username" binding:"required" example:"jane_doe" minLength:"1"`
	Email    string `json:"email" binding:"required" example:"jane@example.com"`
	Bio      string `json:"bio" example:"Updated bio."`
}

// Update godoc
// @Summary      Update a user
// @Description  Replaces the profile fields of an existing user (optimistic lock via version).
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body UpdateUserRequest true "Updated user fields"
// @Success      200  {object}  port.Response{data=entity.User}
// @Router       /users/update [post]
func (h *UserHandler) Update(c *gin.Context) {
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		port.Error(c, port.CodeBadRequest, err.Error())
		return
	}

	user, err := h.uc.Update(c.Request.Context(), req.ID, req.Version, req.Username, req.Email, req.Bio)
	if err != nil {
		port.ResponseError(c, err)
		return
	}

	port.Success(c, user)
}

// DeleteUserRequest is the JSON body for deleting a user.
type DeleteUserRequest struct {
	ID int64 `json:"id" binding:"required" example:"1" minimum:"1"`
}

// Delete godoc
// @Summary      Delete a user
// @Description  Removes a user from the system by their ID.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body DeleteUserRequest true "User ID to delete"
// @Success      200  {object}  port.Response
// @Router       /users/delete [post]
func (h *UserHandler) Delete(c *gin.Context) {
	var req DeleteUserRequest
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
