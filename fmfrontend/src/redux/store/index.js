import thunk from "redux-thunk";
import { createStore, applyMiddleware, combineReducers } from "@reduxjs/toolkit";
import Reducer from "../reducers";

const reducers = combineReducers({
    userReducer: Reducer,
});

const middleware = [thunk];

export const store = createStore(
    reducers,
    applyMiddleware(...middleware)
);


