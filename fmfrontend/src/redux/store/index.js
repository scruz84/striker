import {thunk} from "redux-thunk";
import {applyMiddleware, combineReducers, createStore} from "@reduxjs/toolkit";
import Reducer from "../reducers";

const reducers = combineReducers({
    userReducer: Reducer,
});

const middleware = [thunk];

export const store = createStore(
    reducers,
    applyMiddleware(...middleware)
);


