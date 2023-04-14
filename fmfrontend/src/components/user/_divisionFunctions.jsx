import {divisionResult} from "../../redux/actions";
import {GetApiEndpoint} from "../App";

export function GetDivisionDetails(dispatcher, leagueId) {
    const requestOptions = {
        method: 'GET',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' }
    };
    fetch(GetApiEndpoint('/v1/division/'+leagueId), requestOptions)
        .then(response => {
            return response.json();
        }).then(data => {
        dispatcher(divisionResult(data));
    } );
}
