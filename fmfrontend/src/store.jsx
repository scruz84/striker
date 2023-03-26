import { configureStore } from '@reduxjs/toolkit'
import {Authentication} from "./reducers/authReducer";

export default configureStore({
    reducer: {
        auth: Authentication,
    },
})

