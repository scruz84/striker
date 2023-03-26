import { LOGIN_SUCCESS, LOGOUT, TEAM } from "../types";

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
        case TEAM:
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
        default:
            return initialState;
    }
};