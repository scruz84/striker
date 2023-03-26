import React, {useEffect} from 'react';
import {useDispatch, useSelector} from "react-redux";
import {redirect, useNavigate} from "react-router-dom";
import {GetUserDetails} from "./components/Login";

function Home() {

    const navigate = useNavigate();
    const userReducer = useSelector((state) => state.userReducer);
    const dispatch = useDispatch();

    useEffect(() => {
        // Anything in here is fired on component mount.
        if (!userReducer.loggedUser) {
            GetUserDetails(dispatch, null)
        }
        else {
           navigate('/dashboard');
        }
        return () => {
            // Anything in here is fired on component unmount.
        }
    }, [userReducer.loggedUser])


    return (<div>Welcome to Open Football Manager!</div>);
}

export default Home;
