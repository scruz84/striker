import React, {useEffect} from 'react';
import Home from '../Home';
import {BrowserRouter, Route, Routes} from 'react-router-dom';
import TeamsList from '../TeamsList';
import {ThemeProvider} from '@mui/material/styles';
import {lightTheme} from "../themes/light";
import {Container, CssBaseline} from "@mui/material";
import {CacheProvider} from "@emotion/react";
import createCache from '@emotion/cache';
import Header from "./Header";
import Subheader from "./Subheader";
import {GetUserDetails, Login} from "./Login";
import {GoogleOAuthProvider} from '@react-oauth/google';
import '../App.css';
import {useDispatch, useSelector} from "react-redux";
import {UserDashboard} from "./user/UserDashboard";
import {DivisionDetails} from "./team/DivisionDetails";
import {TeamDetails} from "./team/TeamDetails";

const GOOGLE_OAUTH_CLIENT_ID = isDevelopment() ?
    process.env.REACT_APP_GOOGLE_OAUTH_CLIENT_ID : window.extended.GOOGLE_OAUTH_CLIENT_ID
const API_END_POINT = isDevelopment() ? "" : window.extended.API_ENDPOINT

function App() {
    const dispatch = useDispatch();
    const userReducer = useSelector((state) => state.userReducer);

    useEffect(() => {
        // Anything in here is fired on component mount.
        if (!userReducer.loggedUser) {
            GetUserDetails(dispatch, null)
        }
        return () => {
            // Anything in here is fired on component unmount.
        }
    }, [])

    return (
        <GoogleOAuthProvider clientId={GOOGLE_OAUTH_CLIENT_ID}>
            <CacheProvider
                value={createCache({
                    key: 'css',
                })}>
                <Container maxWidth="xl">
                    <ThemeProvider theme={lightTheme}>
                        <CssBaseline enableColorScheme />
                        <BrowserRouter>
                            <Header/>
                            <Subheader/>
                            <Routes>
                                <Route path="/" element={<Home />}/>
                                <Route path="/allTeams" element={<TeamsList />}/>
                                <Route path="/dashboard" element={<UserDashboard/>}/>
                                <Route path="/division/:id" element={<DivisionDetails/>}/>
                                <Route path="/team/:id" element={<TeamDetails/>}/>
                                <Route path="/login" element={<Login />}/>
                            </Routes>
                        </BrowserRouter>
                    </ThemeProvider>
                </Container>
            </CacheProvider>
        </GoogleOAuthProvider>
    );
}

export default App;

export function GetApiEndpoint(path) {
    console.log("Endpoint: " + API_END_POINT)
    console.log("Path: " + path)
    console.log("Full: " + API_END_POINT + path)
    return API_END_POINT + path
}

function isDevelopment() {
    return !process.env.NODE_ENV || process.env.NODE_ENV === 'development'
}