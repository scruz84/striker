import {
    DIVISION_DETAILS,
    LOGIN_SUCCESS,
    LOGOUT,
    MATCH_DETAILS,
    MATCH_DETAILS_RESET,
    TEAM,
    TEAM_DETAILS,
    TEAM_DETAILS_RESET
} from "../types";

export const loginResult = (loggedUser) => (dispatch, getState) => {
    try {
        if (loggedUser!=null) {
            dispatch({
                type: LOGIN_SUCCESS,
                payload: {
                    loggedUser
                },
            });
        }
        else {
            logOut()
        }

    } catch (error) {
        console.log("Error", error);
    }
};

export const logOut = () => (dispatch, getState) => {
    try {
        dispatch({
            type: LOGOUT,
            payload: { },
        });
    } catch (error) {
        console.log("Error", error);
    }
};

export const teamResult = (userTeam) => (dispatch, getState) => {
    try {
        if (userTeam != null) {
            dispatch({
                type: TEAM,
                payload: {
                    userTeam
                },
            });
        }

    } catch (error) {
        console.log("Error", error);
    }
};

export const divisionResult = (divisionDetails) => (dispatch, getState) => {
    try {
        if (divisionDetails != null) {
            dispatch({
                type: DIVISION_DETAILS,
                payload: {
                    divisionDetails
                },
            });
        }

    } catch (error) {
        console.log("Error getting division details", error);
    }
};

export const teamDetailsResult = (teamDetails) => (dispatch, getState) => {
    try {
        if (teamDetails != null) {
            dispatch({
                type: TEAM_DETAILS,
                payload: {
                    teamDetails
                },
            });
        }

    } catch (error) {
        console.log("Error getting team details", error);
    }
};

export const matchDetailsResult = (matchDetails) => (dispatch, getState) => {
    try {
        if (matchDetails != null) {
            dispatch({
                type: MATCH_DETAILS,
                payload: {
                    matchDetails
                },
            });
        }

    } catch (error) {
        console.log("Error getting match details", error);
    }
};

export const matchDetailsReset = () => (dispatch, getState) => {
    try {
        dispatch({
            type: MATCH_DETAILS_RESET,
            payload: {},
        });
    } catch (error) {
        console.log("Error resetting match details", error);
    }
};


export const teamDetailsReset = () => (dispatch, getState) => {
    try {
        dispatch({
            type: TEAM_DETAILS_RESET,
            payload: {},
        });
    } catch (error) {
        console.log("Error resetting team details", error);
    }
};