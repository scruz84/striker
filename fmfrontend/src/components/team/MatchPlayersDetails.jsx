import SoccerLineUp from 'react-soccer-lineup'
import Box from "@mui/material/Box";
import "./teamview.css";

export function MatchPlayersDetails(matchDetails) {
    let homeTeam = Lineup(true, matchDetails.matchDetails.homeLineup, matchDetails.matchDetails.homePlayers);
    let awayTeam = Lineup(false, matchDetails.matchDetails.awayLineup, matchDetails.matchDetails.awayPlayers)
    return (
        <Box>
            <SoccerLineUp
                size={ "responsive" }
                color={ "#40823d" }
                pattern={ "lines" }
                awayTeam={awayTeam}
                homeTeam={homeTeam}
            />
        </Box>
    )
}

function Lineup(isHome, lineup, players) {
    let color = isHome ? "#f08080" : "#add8e6"
    let gkPos
    let dfPos = []
    let cmPos = []
    let fwPos = []
    let i
    let number = 2
    for (i = 0; i < lineup.positions.length; i++) {
        //goalkeeper
        let pos = lineup.positions[i]
        let player = players[pos.playerId]
        switch (pos.position) {
            case 0:
                gkPos = {number: 1, name: playerName(player.firstName, player.lastName)}
                break;
            case 1:
                dfPos.push({number: number++, name: playerName(player.firstName, player.lastName)})
                break;
            case 2:
                dfPos.push({number: number++, name: playerName(player.firstName, player.lastName)})
                break;
            case 3:
                dfPos.push({number: number++, name: playerName(player.firstName, player.lastName)})
                break;
            case 4:
                cmPos.push({number: number++, name: playerName(player.firstName, player.lastName)})
                break;
            case 5:
                cmPos.push({number: number++, name: playerName(player.firstName, player.lastName)})
                break;
            case 6:
                cmPos.push({number: number++, name: playerName(player.firstName, player.lastName)})
                break;
            case 7:
                fwPos.push({number: number++, name: playerName(player.firstName, player.lastName)})
                break;
            case 8:
                fwPos.push({number: number++, name: playerName(player.firstName, player.lastName)})
                break;
            case 9:
                fwPos.push({number: number++, name: playerName(player.firstName, player.lastName)})
                break;
        }
    }

    return ({
        color: color,
        squad: {
            cm: cmPos,
            df: dfPos,
            fw: fwPos,
            gk: gkPos
        }
    })
}

function playerName(firstName, lastName) {
    return (firstName.charAt(0) + '. ' + lastName)
}
