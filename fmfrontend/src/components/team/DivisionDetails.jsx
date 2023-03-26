import {useParams} from "react-router-dom";

export function DivisionDetails() {
    const routeParams = useParams();

    return (<h2>{routeParams.id}</h2>);
}