import { GoogleLogin } from '@react-oauth/google';
import { useSelector, useDispatch } from 'react-redux';
import {loginResult, logOut} from "../redux/actions";
import {redirect, useNavigate} from "react-router-dom";

export const Login = () => {
    const dispatch = useDispatch()
    const nameReducer = useSelector((state) => state.userReducer);
    const navigate = useNavigate()
    const loggedUser = nameReducer.loggedUser;
    return(
        loggedUser!=null?
            <div>
                Already logged as {loggedUser.name}
            </div>
            : <GoogleLogin
                onSuccess={credentialResponse => {
                    checkCredential(credentialResponse.credential, navigate, dispatch, '/dashboard')
                }}
                onError={() => {
                    console.log('Login Failed');
                }}
            />
    );
}

function checkCredential(credential, navigate, dispatcher, redirectUrl) {
    const requestOptions = {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ jwt: credential, provider: 'google'})
    };
    fetch('/v1/login', requestOptions)
        .then(response => response.json())
        .then(data => {
            GetUserDetails(dispatcher, navigate, redirectUrl)
        } );
}

export function GetUserDetails(dispatcher, navigate, redirectUrl = null) {
    const requestOptions = {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' }
    };
    fetch('/v1/auth/me', requestOptions)
        .then(response => response.json())
        .then(data => {
            if (data.code === 401) {
                dispatcher(logOut());
            } else {
                dispatcher(loginResult(data));
                if (redirectUrl!=null) {
                    navigate(redirectUrl);
                }
            }
        } );
}

export function DoLogout(dispatcher, navigate, redirectUrl = null) {
    const requestOptions = {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
    };
    fetch('/v1/auth/logout', requestOptions)
        .then(response => response.json())
        .then(data => {
            dispatcher(logOut())
            if (redirectUrl!=null) {
                navigate(redirectUrl);
            }
        } );

}
