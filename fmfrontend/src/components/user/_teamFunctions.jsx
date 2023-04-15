import {GetApiEndpoint} from "../App";
import {teamDetailsResult} from "../../redux/actions";

export function GetTeamDetails(dispatcher, teamId) {
    const requestOptions = {
        method: 'GET',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' }
    };
    fetch(GetApiEndpoint('/v1/teams/team/'+teamId+'/'), requestOptions)
        .then(response => {
            return response.json();
        }).then(data => {
        dispatcher(teamDetailsResult(data));
    } );
}
