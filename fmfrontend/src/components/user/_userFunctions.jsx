import {teamResult} from "../../redux/actions";
import {GetApiEndpoint} from "../App";

export function GetUserTeamDetails(dispatcher) {
    const requestOptions = {
        method: 'GET',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' }
    };
    fetch(GetApiEndpoint('/v1/user/team'), requestOptions)
        .then(response => {
                return response.json();
        }).then(data => {
            dispatcher(teamResult(data));
        } );
}
