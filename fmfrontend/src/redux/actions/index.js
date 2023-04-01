import {DIVISION_DETAILS, LOGIN_SUCCESS, LOGOUT, TEAM} from "../types";

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