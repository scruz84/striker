import {teamResult} from "../../redux/actions";

export function GetUserTeamDetails(dispatcher) {
    const requestOptions = {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' }
    };
    fetch('/v1/user/team', requestOptions)
        .then(response => {
                return response.json();
        }).then(data => {
            console.log(data)
            dispatcher(teamResult(data));
        } );
}
