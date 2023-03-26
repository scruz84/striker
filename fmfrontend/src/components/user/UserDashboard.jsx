import {Card, CardContent} from "@mui/material";
import Typography from "@mui/material/Typography";
import {UserTeamDetails} from "./UserTeamDetails";
import {useEffect} from "react";
import {GetUserTeamDetails} from "./_userFunctions";
import {useDispatch, useSelector} from "react-redux";

export function UserDashboard() {
    const dispatch = useDispatch()
    const userReducer = useSelector((state) => state.userReducer);
    const userTeam = userReducer.userTeam;

    useEffect(() => {
        // Anything in here is fired on component mount.
        if (!userTeam) {
            GetUserTeamDetails(dispatch, null)
        }
        return () => {
            // Anything in here is fired on component unmount.
        }
    }, [])

    return (
        userReducer.userTeam?
            <UserTeamDetails userTeam={userReducer.userTeam}/>
            : <div>loading...</div>
    );
}
