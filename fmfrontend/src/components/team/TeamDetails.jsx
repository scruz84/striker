import {Link, useParams} from "react-router-dom";
import {useDispatch, useSelector} from "react-redux";
import React, {useEffect} from "react";
import Box from "@mui/material/Box";
import {GetTeamDetails} from "../user/_teamFunctions";
import {teamDetailsReset} from "../../redux/actions";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";

export function TeamDetails() {
    const routeParams = useParams();

    const dispatch = useDispatch()
    const teamReducer = useSelector((state) => state.userReducer);
    const teamDetails = teamReducer.teamDetails;

    useEffect(() => {
        if (!teamDetails) {
            GetTeamDetails(dispatch, routeParams.id)
        }
        return () => {
            dispatch(teamDetailsReset());
        }
    }, [])

    if (teamReducer.teamDetails) {
        console.log(teamReducer.teamDetails)
        return ( _teamDetails(teamReducer.teamDetails) );
    } else {
        return ( <div>loading...</div> );
    }
}

function _teamDetails(teamDetails) {
   return (
       <Stack>
           <Typography variant="h4" gutterBottom>{teamDetails.team.name} ({teamDetails.team.id})</Typography>
            <Box>
                <Typography sx={{fontWeight: 'bold'}} gutterBottom>Competiciones:</Typography>
                <Typography variant="body1">
                    <Link to={`/division/${teamDetails.team.divisionId}`}> {teamDetails.team.divisionName}</Link>
                </Typography>
            </Box>
       </Stack>
   );
}