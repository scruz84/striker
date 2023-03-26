import { LOGIN_SUCCESS, LOGOUT, TEAM } from "../types";

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