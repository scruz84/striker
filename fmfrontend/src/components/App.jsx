import React, {useEffect} from 'react';
import Home from '../Home';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import TeamsList from '../TeamsList';
import ClientEdit from "../ClientEdit";
import { red } from '@mui/material/colors';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import { lightTheme } from "../themes/light";
import {Container, CssBaseline} from "@mui/material";
import {CacheProvider} from "@emotion/react";
import createCache from '@emotion/cache';
import PropTypes from "prop-types";
import Header from "./Header";
import Subheader from "./Subheader";
import {Login, GetUserDetails} from "./Login";
import { GoogleOAuthProvider } from '@react-oauth/google';
import '../App.css';
import {useDispatch, useSelector} from "react-redux";
import {UserDashboard} from "./user/UserDashboard";
import {DivisionDetails} from "./team/DivisionDetails";

const GOOGLE_OAUTH_CLIENT_ID = process.env.REACT_APP_GOOGLE_OAUTH_CLIENT_ID;

function App() {
    const dispatch = useDispatch()
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
