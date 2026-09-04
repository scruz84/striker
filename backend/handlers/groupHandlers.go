package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/scruz84/striker/common"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type CreateGroupRequest struct {
	Name string `json:"name"`
}

// CreateGroupHandler makes a new friend group with the caller as its owner and first member.
func CreateGroupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		var req CreateGroupRequest
		if err := c.BindJSON(&req); err != nil || req.Name == "" {
			c.JSON(http.StatusBadRequest, GetError(0, "name is required"))
			return
		}

		group, err := common.CreateGroup(dbCtx, req.Name, *userId)
		if err != nil {
			log.Errorf("error creating group for user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, group)
	}
}

type JoinGroupRequest struct {
	InviteCode string `json:"inviteCode"`
}

// JoinGroupHandler adds the caller to a group by invite code.
func JoinGroupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		var req JoinGroupRequest
		if err := c.BindJSON(&req); err != nil || req.InviteCode == "" {
			c.JSON(http.StatusBadRequest, GetError(0, "inviteCode is required"))
			return
		}

		group, err := common.JoinGroup(dbCtx, req.InviteCode, *userId)
		if err != nil {
			switch {
			case errors.Is(err, common.ErrInvalidInviteCode):
				c.JSON(http.StatusNotFound, GetError(0, err.Error()))
			case errors.Is(err, common.ErrAlreadyInGroup):
				c.JSON(http.StatusConflict, GetError(0, err.Error()))
			default:
				log.Errorf("error joining group for user %d: %s", *userId, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			}
			return
		}

		c.JSON(http.StatusOK, group)
	}
}

// MyGroupsHandler lists every group the caller belongs to.
func MyGroupsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		groups, err := common.GetGroupsForUser(dbCtx, *userId)
		if err != nil {
			log.Errorf("error listing groups for user %d: %s", *userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, groups)
	}
}

type GroupDetailsResponse struct {
	Group       common.Group         `json:"group"`
	Leaderboard []common.GroupMember `json:"leaderboard"`
}

// GroupDetailsHandler returns a group and its leaderboard - members only, so a group's
// composition/finances stay visible only to people who joined it.
func GroupDetailsHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		groupId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid group id"))
			return
		}

		isMember, err := common.IsGroupMember(dbCtx, groupId, *userId)
		if err != nil {
			log.Errorf("error checking group membership for user %d group %d: %s", *userId, groupId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}
		if !isMember {
			c.JSON(http.StatusForbidden, GetError(0, "not a member of this group"))
			return
		}

		group, err := common.GetGroupById(dbCtx, groupId)
		if err != nil {
			if errors.Is(err, common.ErrGroupNotFound) {
				c.JSON(http.StatusNotFound, GetError(0, err.Error()))
				return
			}
			log.Errorf("error getting group %d: %s", groupId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		leaderboard, err := common.GetGroupLeaderboard(dbCtx, groupId)
		if err != nil {
			log.Errorf("error getting leaderboard for group %d: %s", groupId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, GroupDetailsResponse{Group: *group, Leaderboard: leaderboard})
	}
}

// LeaveGroupHandler removes the caller from a group.
func LeaveGroupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		groupId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid group id"))
			return
		}

		if err := common.LeaveGroup(dbCtx, groupId, *userId); err != nil {
			log.Errorf("error leaving group %d for user %d: %s", groupId, *userId, err)
			c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "left group"})
	}
}

// DeleteGroupHandler deletes a group entirely - owner only.
func DeleteGroupHandler(dbCtx *common.DatabaseContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := GetUserIdFromCookie(c)
		if userId == nil {
			c.JSON(http.StatusForbidden, nil)
			return
		}

		groupId, err := common.StringToInt64(c.Param(ID))
		if err != nil {
			c.JSON(http.StatusBadRequest, GetError(0, "invalid group id"))
			return
		}

		if err := common.DeleteGroup(dbCtx, groupId, *userId); err != nil {
			switch {
			case errors.Is(err, common.ErrGroupNotFound):
				c.JSON(http.StatusNotFound, GetError(0, err.Error()))
			case errors.Is(err, common.ErrNotGroupOwner):
				c.JSON(http.StatusForbidden, GetError(0, err.Error()))
			default:
				log.Errorf("error deleting group %d: %s", groupId, err)
				c.JSON(http.StatusInternalServerError, GetError(0, err.Error()))
			}
			return
		}

		c.JSON(http.StatusOK, common.HttpResponse{Message: "group deleted"})
	}
}
