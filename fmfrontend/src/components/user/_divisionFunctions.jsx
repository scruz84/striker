import {divisionResult} from "../../redux/actions";

export function GetDivisionDetails(dispatcher, leagueId) {
    const requestOptions = {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' }
    };
    fetch('/v1/division/'+leagueId, requestOptions)
        .then(response => {
            return response.json();
        }).then(data => {
        console.log(data)
        dispatcher(divisionResult(data));
    } );
}
