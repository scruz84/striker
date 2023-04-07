import {DIVISION_DETAILS, LOGIN_SUCCESS, LOGOUT, TEAM, TEAM_DETAILS, TEAM_DETAILS_RESET} from "../types";

const initialState = {
    id:null,
    name:null,
    provider:null,
    email:null,
    country:null,
    teamId :null,
    teamName:null,
    teamLeague:null,
    pictureUrl:null,
}

export default (state = initialState, action) => {
    switch (action.type) {
        case DIVISION_DETAILS:
        case TEAM:
        case TEAM_DETAILS:
        case LOGIN_SUCCESS:
            return {
                ...state,
                ...action.payload,
            };
        case LOGOUT:
            return {
                ...{},
                ...action.payload,
            };
        case TEAM_DETAILS_RESET:
            state.teamDetails = null
            return {
                ...state,
            }
        default:
            return initialState;
    }
};