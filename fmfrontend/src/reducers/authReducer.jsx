import { appConstants } from '../constants/constants';

const initialState = {
    Id:null,
    Name:null,
    Provider:null,
    Email:null,
    Country:null,
    TeamId :null,
    TeamName:null,
    TeamLeague:null,
    PictureUrl:null,
}

export function Authentication(state = initialState, action) {
    switch (action) {
        case appConstants.LOGIN_SUCCESS:
            return {
                type: 'LOGIN_SUCCESS',
                payload: state
            };
        case appConstants.LOGOUT:
            return {};
        default:
            return state
    }
}
